package ingest

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/likaia/nginxpulse/internal/enrich"
	"github.com/likaia/nginxpulse/internal/store"
	"github.com/sirupsen/logrus"
)

type parseSourceContext struct {
	sourceID    string
	sourceKey   string
	startOffset int64
	hasOffset   bool
}

func (p *LogParser) scanSingleFile(
	websiteID string, logPath string, parserResult *ParserResult) {
	file, err := os.Open(logPath)
	if err != nil {
		logrus.Errorf("无法打开日志文件 %s: %v", logPath, err)
		p.notifyFileIO(websiteID, logPath, "打开日志文件", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		logrus.Errorf("无法获取文件信息 %s: %v", logPath, err)
		p.notifyFileIO(websiteID, logPath, "读取日志文件信息", err)
		return
	}

	currentSize := fileInfo.Size()
	isGzip := isGzipFile(logPath)

	parser, err := p.getLineParser(websiteID)
	if err != nil {
		parserResult.Success = false
		parserResult.Error = err
		p.notifyLogParsing(websiteID, logPath, "日志解析配置", err)
		return
	}

	fileState, ok := p.getFileState(websiteID, logPath)
	if ok && currentSize < fileState.LastSize {
		logrus.Infof("检测到网站 %s 的日志文件 %s 已被轮转，从头开始扫描", websiteID, logPath)
		ok = false
		p.deleteFileState(websiteID, logPath)
	}

	if !ok {
		fileState = FileState{}
		cutoff := time.Now().AddDate(0, 0, -recentLogWindowDays)
		cutoffTs := cutoff.Unix()
		fileState.RecentCutoffTs = cutoffTs

		p.initFileRange(file, parser, fileInfo, isGzip, &fileState)

		if isGzip {
			if fileInfo.ModTime().After(cutoff) || fileInfo.ModTime().Equal(cutoff) {
				if _, err := file.Seek(0, 0); err == nil {
					if gzReader, err := gzip.NewReader(file); err == nil {
						sourceCtx := fileParseSourceContext(logPath, 0)
						entriesCount, _, minTs, maxTs := p.parseLogLines(
							gzReader, websiteID, sourceCtx, parserResult, parseWindow{minTs: cutoffTs},
						)
						gzReader.Close()
						p.updateParsedRange(&fileState, minTs, maxTs)
						if maxTs > fileState.LastTimestamp {
							fileState.LastTimestamp = maxTs
						}
						if entriesCount > 0 {
							logrus.Infof("网站 %s 的 gzip 日志文件 %s 扫描完成，解析了 %d 条记录",
								websiteID, logPath, entriesCount)
						}
					} else {
						logrus.Errorf("无法解析 gzip 日志文件 %s: %v", logPath, err)
						p.notifyLogParsing(websiteID, logPath, "解析 gzip 日志文件", err)
					}
				} else {
					logrus.Errorf("无法重置 gzip 文件 %s: %v", logPath, err)
					p.notifyFileIO(websiteID, logPath, "重置 gzip 文件指针", err)
				}
			}

			fileState.LastSize = currentSize
			fileState.LastOffset = 0
			fileState.BackfillOffset = 0
			fileState.BackfillEnd = 0
			fileState.BackfillDone = fileState.FirstTimestamp > 0 && fileState.FirstTimestamp >= cutoffTs
			p.setFileState(websiteID, logPath, fileState)
			return
		}

		recentOffset, lastTs, err := p.findRecentOffset(file, parser, cutoff)
		backfillEnd := recentOffset
		if err != nil {
			logrus.Warnf("计算日志文件 %s 最近窗口失败: %v", logPath, err)
			p.notifyFileIO(websiteID, logPath, "计算日志文件最近窗口", err)
			backfillEnd = currentSize
			recentOffset = 0
		}
		if lastTs > 0 {
			fileState.LastTimestamp = lastTs
		}
		fileState.RecentOffset = recentOffset
		fileState.BackfillOffset = 0
		fileState.BackfillEnd = backfillEnd
		fileState.BackfillDone = err == nil && recentOffset == 0
		fileState.LastOffset = currentSize
		fileState.LastSize = currentSize

		if recentOffset < currentSize {
			if _, err := file.Seek(recentOffset, 0); err != nil {
				logrus.Errorf("无法设置文件读取位置 %s: %v", logPath, err)
				p.notifyFileIO(websiteID, logPath, "设置文件读取位置", err)
			} else {
				sourceCtx := fileParseSourceContext(logPath, recentOffset)
				entriesCount, bytesRead, minTs, maxTs := p.parseLogLines(
					file, websiteID, sourceCtx, parserResult, parseWindow{minTs: cutoffTs},
				)
				fileState.LastOffset = recentOffset + bytesRead
				p.updateParsedRange(&fileState, minTs, maxTs)
				if maxTs > fileState.LastTimestamp {
					fileState.LastTimestamp = maxTs
				}
				if entriesCount > 0 {
					logrus.Infof("网站 %s 的日志文件 %s 扫描完成，解析了 %d 条记录",
						websiteID, logPath, entriesCount)
				}
			}
		}

		p.setFileState(websiteID, logPath, fileState)
		return
	}

	startOffset := p.determineStartOffset(websiteID, logPath, currentSize)
	if startOffset < 0 {
		return
	}
	if !isGzip && currentSize <= startOffset {
		return
	}

	var (
		reader io.Reader
		closer io.Closer
	)
	if isGzip {
		if _, err = file.Seek(0, 0); err != nil {
			logrus.Errorf("无法设置文件读取位置 %s: %v", logPath, err)
			p.notifyFileIO(websiteID, logPath, "设置文件读取位置", err)
			return
		}
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			logrus.Errorf("无法解析 gzip 日志文件 %s: %v", logPath, err)
			p.notifyLogParsing(websiteID, logPath, "解析 gzip 日志文件", err)
			return
		}
		if startOffset > 0 {
			if err := skipReaderBytes(gzReader, startOffset); err != nil {
				logrus.Warnf("跳过 gzip 历史内容失败，将重新解析文件 %s: %v", logPath, err)
				gzReader.Close()
				if _, err := file.Seek(0, 0); err != nil {
					logrus.Errorf("无法重置 gzip 文件 %s: %v", logPath, err)
					p.notifyFileIO(websiteID, logPath, "重置 gzip 文件指针", err)
					return
				}
				gzReader, err = gzip.NewReader(file)
				if err != nil {
					logrus.Errorf("无法重新解析 gzip 日志文件 %s: %v", logPath, err)
					p.notifyLogParsing(websiteID, logPath, "重新解析 gzip 日志文件", err)
					return
				}
				startOffset = 0
			}
		}
		reader = gzReader
		closer = gzReader
	} else {
		if _, err = file.Seek(startOffset, 0); err != nil {
			logrus.Errorf("无法设置文件读取位置 %s: %v", logPath, err)
			p.notifyFileIO(websiteID, logPath, "设置文件读取位置", err)
			return
		}
		reader = file
	}

	sourceCtx := fileParseSourceContext(logPath, startOffset)
	entriesCount, bytesRead, minTs, maxTs := p.parseLogLines(reader, websiteID, sourceCtx, parserResult, parseWindow{})
	if closer != nil {
		closer.Close()
	}

	fileState.LastOffset = startOffset + bytesRead
	fileState.LastSize = currentSize
	p.updateParsedRange(&fileState, minTs, maxTs)
	if maxTs > fileState.LastTimestamp {
		fileState.LastTimestamp = maxTs
	}

	p.setFileState(websiteID, logPath, fileState)

	if entriesCount > 0 {
		logrus.Infof("网站 %s 的日志文件 %s 扫描完成，解析了 %d 条记录",
			websiteID, logPath, entriesCount)
	}
}

// determineStartOffset 确定扫描起始位置
func (p *LogParser) determineStartOffset(
	websiteID string, filePath string, currentSize int64) int64 {

	state, ok := p.states[websiteID]
	if !ok { // 网站没有扫描记录，创建新状态
		p.states[websiteID] = LogScanState{
			Files: make(map[string]FileState),
		}
		return 0
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
		p.states[websiteID] = state
		return 0
	}

	normalizedPath := normalizeLogPath(filePath)
	fileState, ok := state.Files[normalizedPath]
	if !ok {
		return 0
	}

	// 文件是否被轮转
	if currentSize < fileState.LastSize {
		logrus.Infof("检测到网站 %s 的日志文件 %s 已被轮转，从头开始扫描", websiteID, filePath)
		return 0
	}

	if isGzipFile(filePath) {
		if currentSize == fileState.LastSize {
			return -1
		}
		return fileState.LastOffset
	}

	return fileState.LastOffset
}

func (p *LogParser) initFileRange(
	file *os.File,
	parser *logLineParser,
	info os.FileInfo,
	isGzip bool,
	state *FileState,
) {
	if state.FirstTimestamp == 0 {
		if firstTs, err := p.readFirstTimestamp(file, parser, isGzip); err == nil {
			state.FirstTimestamp = firstTs
		}
	}
	if state.LastTimestamp == 0 {
		state.LastTimestamp = info.ModTime().Unix()
	}
}

func (p *LogParser) updateParsedRange(state *FileState, minTs, maxTs int64) {
	if minTs > 0 && (state.ParsedMinTs == 0 || minTs < state.ParsedMinTs) {
		state.ParsedMinTs = minTs
	}
	if maxTs > 0 && maxTs > state.ParsedMaxTs {
		state.ParsedMaxTs = maxTs
	}
	if state.FirstTimestamp == 0 || (minTs > 0 && minTs < state.FirstTimestamp) {
		state.FirstTimestamp = minTs
	}
	if maxTs > 0 && maxTs > state.LastTimestamp {
		state.LastTimestamp = maxTs
	}
}

func (p *LogParser) readFirstTimestamp(
	file *os.File,
	parser *logLineParser,
	isGzip bool,
) (int64, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return 0, err
	}

	var reader io.Reader = file
	var closer io.Closer
	if isGzip {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return 0, err
		}
		reader = gzReader
		closer = gzReader
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ts, err := p.parseLogTimestamp(parser, line)
		if err == nil {
			if closer != nil {
				closer.Close()
			}
			return ts.Unix(), nil
		}
	}

	if closer != nil {
		closer.Close()
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("未找到有效的日志时间")
}

func (p *LogParser) findRecentOffset(
	file *os.File,
	parser *logLineParser,
	cutoff time.Time,
) (int64, int64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}
	size := info.Size()
	if size == 0 {
		return 0, 0, nil
	}

	var (
		offset  = size
		carry   []byte
		lastTs  int64
		started bool
	)

	for offset > 0 {
		readSize := int64(recentScanChunkSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		buf := make([]byte, readSize)
		if _, err := file.ReadAt(buf, offset); err != nil && err != io.EOF {
			return 0, lastTs, err
		}

		data := append(buf, carry...)
		start := 0
		if offset > 0 {
			if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
				carry = append([]byte{}, data[:idx]...)
				start = idx + 1
			} else {
				carry = append([]byte{}, data...)
				continue
			}
		} else {
			carry = nil
		}

		end := len(data)
		for end > start {
			lineEnd := end
			idx := bytes.LastIndexByte(data[start:end], '\n')
			lineStart := start
			if idx >= 0 {
				lineStart = start + idx + 1
				end = start + idx
			} else {
				end = start
			}
			line := bytes.TrimRight(data[lineStart:lineEnd], "\r")
			if len(line) == 0 {
				continue
			}
			ts, err := p.parseLogTimestamp(parser, string(line))
			if err != nil {
				continue
			}
			if !started {
				lastTs = ts.Unix()
				started = true
			}
			if ts.Before(cutoff) {
				nextOffset := offset + int64(lineEnd)
				if lineEnd < len(data) && data[lineEnd] == '\n' {
					nextOffset++
				}
				if nextOffset > size {
					nextOffset = size
				}
				return nextOffset, lastTs, nil
			}
		}
		if offset == 0 {
			break
		}
	}

	return 0, lastTs, nil
}

func readLineAlignedShards(
	reader io.Reader,
	targetSize int,
	emit func(baseBytes int64, data []byte) bool,
) (int64, error) {
	if targetSize <= 0 {
		targetSize = 256 * 1024
	}
	buffered := bufio.NewReaderSize(reader, targetSize)
	shard := make([]byte, 0, targetSize)
	var shardBase int64
	var totalBytes int64

	for {
		line, err := buffered.ReadBytes('\n')
		if len(line) > 0 {
			shard = append(shard, line...)
			totalBytes += int64(len(line))
		}
		if len(shard) >= targetSize {
			if !emit(shardBase, shard) {
				return totalBytes, nil
			}
			shardBase = totalBytes
			shard = make([]byte, 0, targetSize)
		}
		if err != nil {
			if len(shard) > 0 {
				_ = emit(shardBase, shard)
			}
			if errors.Is(err, io.EOF) {
				return totalBytes, nil
			}
			return totalBytes, err
		}
	}
}

// parseLogLines 解析日志行，并将多 worker 字节分片解析与批量写入组成有界流水线。
func (p *LogParser) parseLogLines(
	reader io.Reader, websiteID string, sourceCtx parseSourceContext, parserResult *ParserResult, window parseWindow) (int, int64, int64, int64) {
	type batchJob struct {
		logs          []store.NginxLogRecord
		whitelistHits map[string]*whitelistHit
		endBytes      int64
		minTs         int64
		maxTs         int64
		buckets       map[int64]struct{}
	}
	type batchResult struct {
		job batchJob
		err error
	}

	type parseShardJob struct {
		seq       int64
		baseBytes int64
		data      []byte
	}
	type parsedShardEntry struct {
		entry    store.NginxLogRecord
		match    enrich.WhitelistMatch
		matched  bool
		endBytes int64
	}
	type parseShardResult struct {
		seq     int64
		entries []parsedShardEntry
	}
	type scanSummary struct {
		totalBytes int64
		err        error
	}

	lineParser, err := p.getLineParserForSource(websiteID, sourceCtx.sourceID)
	if err != nil {
		parserResult.Success = false
		parserResult.Error = err
		return 0, 0, 0, 0
	}

	entriesCount := 0
	var minTs int64
	var maxTs int64
	parsedBuckets := make(map[int64]struct{})
	var whitelistHits map[string]*whitelistHit
	domainMatcher := newWebsiteDomainMatcher(websiteID)
	whitelistMatcher := p.whitelistMatchers[websiteID]

	// 只允许一个批次在数据库中执行。主 goroutine 在此期间解析下一批，既实现重叠，
	// 又避免数据库变慢时未落库批次和内存无界增长。
	jobs := make(chan batchJob)
	results := make(chan batchResult, 1)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for job := range jobs {
			p.markBatchIPGeoPending(job.logs)
			err := p.repo.BatchInsertLogsForWebsite(websiteID, job.logs)
			if err == nil {
				p.enqueueBatchIPGeo(job.logs)
			}
			results <- batchResult{job: job, err: err}
		}
	}()
	defer func() {
		close(jobs)
		<-writerDone
	}()

	batch := make([]store.NginxLogRecord, 0, p.parseBatchSize)
	var batchWhitelistHits map[string]*whitelistHit
	var batchMinTs int64
	var batchMaxTs int64
	batchBuckets := make(map[int64]struct{})
	var committedBytes int64
	inFlight := false
	pipelineFailed := false

	mergeSuccessfulBatch := func(result batchResult) bool {
		inFlight = false
		if result.err != nil {
			parserResult.Success = false
			parserResult.Error = result.err
			logrus.Errorf("批量插入网站 %s 的日志记录失败: %v", websiteID, result.err)
			p.notifyDatabaseWrite(websiteID, "写入日志批次", result.err)
			return false
		}

		committedBytes = result.job.endBytes
		entriesCount += len(result.job.logs)
		parserResult.TotalEntries += len(result.job.logs)
		whitelistHits = mergeWhitelistHits(whitelistHits, result.job.whitelistHits)
		if result.job.minTs > 0 && (minTs == 0 || result.job.minTs < minTs) {
			minTs = result.job.minTs
		}
		if result.job.maxTs > maxTs {
			maxTs = result.job.maxTs
		}
		for bucket := range result.job.buckets {
			parsedBuckets[bucket] = struct{}{}
		}
		return true
	}

	awaitInFlight := func() bool {
		if !inFlight {
			return true
		}
		return mergeSuccessfulBatch(<-results)
	}

	submitBatch := func(endBytes int64) bool {
		if len(batch) == 0 {
			return true
		}
		if !awaitInFlight() {
			return false
		}

		jobs <- batchJob{
			logs:          batch,
			whitelistHits: batchWhitelistHits,
			endBytes:      endBytes,
			minTs:         batchMinTs,
			maxTs:         batchMaxTs,
			buckets:       batchBuckets,
		}
		inFlight = true
		batch = make([]store.NginxLogRecord, 0, p.parseBatchSize)
		batchWhitelistHits = nil
		batchMinTs = 0
		batchMaxTs = 0
		batchBuckets = make(map[int64]struct{})
		return true
	}

	const (
		parseShardSize = 256 * 1024
		maxParseWorker = 32
	)
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > maxParseWorker {
		workerCount = maxParseWorker
	}
	queueSize := workerCount * 2
	parseJobs := make(chan parseShardJob, queueSize)
	parseResults := make(chan parseShardResult, queueSize)
	shardSlots := make(chan struct{}, queueSize)
	scanDone := make(chan scanSummary, 1)
	cancelParsing := make(chan struct{})
	var cancelOnce sync.Once
	stopParsing := func() {
		cancelOnce.Do(func() { close(cancelParsing) })
	}
	defer stopParsing()

	// 每个 worker 独立遍历一个换行对齐的字节分片，并执行正则、域名、时间窗口和白名单过滤。
	parseShard := func(job parseShardJob) parseShardResult {
		result := parseShardResult{
			seq:     job.seq,
			entries: make([]parsedShardEntry, 0, len(job.data)/256),
		}
		for cursor := 0; cursor < len(job.data); {
			lineEnd := len(job.data)
			next := len(job.data)
			if newline := bytes.IndexByte(job.data[cursor:], '\n'); newline >= 0 {
				lineEnd = cursor + newline
				next = lineEnd + 1
			}
			contentEnd := lineEnd
			if contentEnd > cursor && job.data[contentEnd-1] == '\r' {
				contentEnd--
			}
			line := string(job.data[cursor:contentEnd])
			lineOffset := sourceCtx.startOffset + job.baseBytes + int64(cursor)
			endBytes := job.baseBytes + int64(next)
			cursor = next

			var entry *store.NginxLogRecord
			var err error
			switch lineParser.parseType {
			case parseTypeCaddyJSON:
				entry, err = p.parseCaddyJSONLine(line, lineParser)
			default:
				entry, err = p.parseRegexLogLine(lineParser, line)
			}
			if err != nil || !domainMatcher.includesHost(entry.Host) {
				continue
			}
			if !window.allows(entry.Timestamp.Unix()) {
				continue
			}
			entry.Fingerprint = buildLogLineFingerprint(sourceCtx, lineOffset, line)
			parsed := parsedShardEntry{entry: *entry, endBytes: endBytes}
			if whitelistMatcher != nil && whitelistMatcher.Enabled() {
				parsed.match, parsed.matched = whitelistMatcher.Match(entry.IP)
			}
			result.entries = append(result.entries, parsed)
		}
		return result
	}

	var parseWG sync.WaitGroup
	parseWG.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer parseWG.Done()
			for job := range parseJobs {
				result := parseShard(job)
				select {
				case parseResults <- result:
				case <-cancelParsing:
					return
				}
			}
		}()
	}

	// 读取端以目标字节数切分，但只在完整换行之后提交 shard，绝不截断日志行。
	go func() {
		var seq int64
		dispatchShard := func(base int64, data []byte) bool {
			if len(data) == 0 {
				return true
			}
			select {
			case shardSlots <- struct{}{}:
			case <-cancelParsing:
				return false
			}
			select {
			case parseJobs <- parseShardJob{seq: seq, baseBytes: base, data: data}:
				seq++
				addParsingProgress(int64(len(data)))
				return true
			case <-cancelParsing:
				return false
			}
		}

		totalBytes, readErr := readLineAlignedShards(reader, parseShardSize, dispatchShard)
		close(parseJobs)
		parseWG.Wait()
		close(parseResults)
		scanDone <- scanSummary{totalBytes: totalBytes, err: readErr}
	}()

	// shard 完成顺序不确定；按字节范围序号重排后再组批，保持原日志顺序和精确 offset。
	nextSeq := int64(0)
	pendingResults := make(map[int64]parseShardResult, queueSize)
	for result := range parseResults {
		if pipelineFailed {
			continue
		}
		pendingResults[result.seq] = result
		for {
			ordered, ok := pendingResults[nextSeq]
			if !ok {
				break
			}
			delete(pendingResults, nextSeq)
			nextSeq++
			<-shardSlots

			for _, parsed := range ordered.entries {
				entry := parsed.entry
				ts := entry.Timestamp.Unix()
				if parsed.matched {
					batchWhitelistHits = p.recordWhitelistHit(websiteID, entry, parsed.match, batchWhitelistHits)
				}
				batch = append(batch, entry)
				bucket := (ts / 3600) * 3600
				batchBuckets[bucket] = struct{}{}
				if batchMinTs == 0 || ts < batchMinTs {
					batchMinTs = ts
				}
				if ts > batchMaxTs {
					batchMaxTs = ts
				}

				if len(batch) >= p.parseBatchSize {
					if !submitBatch(parsed.endBytes) {
						pipelineFailed = true
						stopParsing()
						break
					}
				}
			}
			if pipelineFailed {
				break
			}
		}
	}

	summary := <-scanDone
	if !pipelineFailed {
		if submitBatch(summary.totalBytes) {
			if !awaitInFlight() {
				pipelineFailed = true
			}
		} else {
			pipelineFailed = true
		}
	}

	if summary.err != nil {
		parserResult.Success = false
		parserResult.Error = summary.err
		logrus.Errorf("扫描网站 %s 的文件时出错: %v", websiteID, summary.err)
		p.notifyLogParsing(websiteID, "", "扫描日志文件", summary.err)
	}
	p.flushWhitelistHits(whitelistHits)
	p.recordParsedHourBuckets(websiteID, parsedBuckets)

	// 全部批次成功时，可以安全跳过末尾被过滤/无法解析的行；数据库失败时只推进到
	// 最后一个已提交批次，确保失败批次会在下次扫描时重试。
	if !pipelineFailed {
		committedBytes = summary.totalBytes
	}
	return entriesCount, committedBytes, minTs, maxTs
}

// IngestLines parses and inserts streamed log lines for a website/source.
func (p *LogParser) IngestLines(websiteID, sourceID string, lines []string) (int, int, error) {
	if websiteID == "" {
		return 0, 0, errors.New("websiteID 不能为空")
	}
	if len(lines) == 0 {
		return 0, 0, nil
	}
	if _, err := p.getLineParserForSource(websiteID, sourceID); err != nil {
		return 0, 0, err
	}

	batch := make([]store.NginxLogRecord, 0, p.parseBatchSize)
	accepted := 0
	deduped := 0
	var minTs int64
	var maxTs int64
	parsedBuckets := make(map[int64]struct{})
	var whitelistHits map[string]*whitelistHit
	var batchWhitelistHits map[string]*whitelistHit
	domainMatcher := newWebsiteDomainMatcher(websiteID)

	processBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		// 先标记 location 为“待解析”，再在成功落库后写入 ip_geo_pending（避免竞态导致“待解析”长期不变）
		p.markBatchIPGeoPending(batch)
		if err := p.repo.BatchInsertLogsForWebsite(websiteID, batch); err != nil {
			p.notifyDatabaseWrite(websiteID, "写入日志批次", err)
			return err
		}
		p.enqueueBatchIPGeo(batch)
		whitelistHits = mergeWhitelistHits(whitelistHits, batchWhitelistHits)
		batch = batch[:0]
		batchWhitelistHits = nil
		return nil
	}

	for _, line := range lines {
		entry, err := p.parseLogLine(websiteID, sourceID, line)
		if err != nil {
			continue
		}
		if !domainMatcher.includesHost(entry.Host) {
			continue
		}
		entry.Fingerprint = streamLogLineFingerprint(sourceID, line)
		key := buildDedupKey(websiteID, sourceID, line)
		if p.dedup != nil && p.dedup.Seen(key) {
			deduped++
			continue
		}
		if matcher := p.whitelistMatchers[websiteID]; matcher != nil && matcher.Enabled() {
			if match, ok := matcher.Match(entry.IP); ok {
				batchWhitelistHits = p.recordWhitelistHit(websiteID, *entry, match, batchWhitelistHits)
			}
		}
		batch = append(batch, *entry)
		accepted++
		ts := entry.Timestamp.Unix()
		bucket := (ts / 3600) * 3600
		parsedBuckets[bucket] = struct{}{}
		if minTs == 0 || ts < minTs {
			minTs = ts
		}
		if ts > maxTs {
			maxTs = ts
		}

		if len(batch) >= p.parseBatchSize {
			if err := processBatch(); err != nil {
				return accepted, deduped, err
			}
		}
	}

	if err := processBatch(); err != nil {
		return accepted, deduped, err
	}
	p.flushWhitelistHits(whitelistHits)

	if accepted > 0 {
		p.recordParsedHourBuckets(websiteID, parsedBuckets)
		targetKey := buildTargetStateKey(sourceID, "stream")
		state, _ := p.getTargetState(websiteID, targetKey)
		if state.RecentCutoffTs == 0 {
			state.RecentCutoffTs = time.Now().AddDate(0, 0, -recentLogWindowDays).Unix()
		}
		updateTargetParsedRange(&state, minTs, maxTs)
		state.BackfillDone = true
		p.setTargetState(websiteID, targetKey, state)
		p.refreshWebsiteRanges(websiteID)
		p.updateState()
	}

	return accepted, deduped, nil
}

func buildDedupKey(websiteID, sourceID, line string) string {
	hash := sha1.Sum([]byte(line))
	if sourceID == "" {
		return fmt.Sprintf("%s:%x", websiteID, hash[:])
	}
	return fmt.Sprintf("%s:%s:%x", websiteID, sourceID, hash[:])
}

func fileParseSourceContext(logPath string, startOffset int64) parseSourceContext {
	return parseSourceContext{
		sourceKey:   normalizeLogPath(logPath),
		startOffset: startOffset,
		hasOffset:   true,
	}
}

func targetParseSourceContext(sourceID, targetKey string, startOffset int64) parseSourceContext {
	return parseSourceContext{
		sourceID:    sourceID,
		sourceKey:   buildTargetStateKey(sourceID, targetKey),
		startOffset: startOffset,
		hasOffset:   true,
	}
}

func streamLogLineFingerprint(sourceID, line string) string {
	hash := sha1.Sum([]byte("stream:v1\x00" + sourceID + "\x00" + line))
	return fmt.Sprintf("%x", hash[:])
}

func buildLogLineFingerprint(sourceCtx parseSourceContext, lineOffset int64, line string) string {
	if sourceCtx.hasOffset {
		hash := sha1.Sum([]byte(fmt.Sprintf("offset:v1\x00%s\x00%d\x00%s", sourceCtx.sourceKey, lineOffset, line)))
		return fmt.Sprintf("%x", hash[:])
	}
	return streamLogLineFingerprint(sourceCtx.sourceID, line)
}

// markBatchIPGeoPending mutates the batch in-place to mark locations as "待解析"/"未知".
// 注意：该操作必须发生在日志入库之前，否则日志不会以“待解析”维度写入。
func (p *LogParser) markBatchIPGeoPending(batch []store.NginxLogRecord) {
	if len(batch) == 0 {
		return
	}
	for i := range batch {
		ip := strings.TrimSpace(batch[i].IP)
		if ip == "" {
			batch[i].DomesticLocation = "未知"
			batch[i].GlobalLocation = "未知"
			continue
		}
		batch[i].DomesticLocation = pendingLocationLabel
		batch[i].GlobalLocation = pendingLocationLabel
	}
}

// enqueueBatchIPGeo writes unique IPs from the batch into ip_geo_pending.
// 注意：该操作应在日志成功落库之后再执行，避免“先入队、后落库”导致回填命中空结果并清理 pending，进而让日志长期停留在“待解析”。
func (p *LogParser) enqueueBatchIPGeo(batch []store.NginxLogRecord) {
	if len(batch) == 0 || p.repo == nil || p.demoMode {
		return
	}
	unique := make([]string, 0, len(batch))
	seen := make(map[string]struct{}, len(batch))
	for _, entry := range batch {
		ip := strings.TrimSpace(entry.IP)
		if ip == "" {
			continue
		}
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = struct{}{}
		unique = append(unique, ip)
	}
	if len(unique) == 0 {
		return
	}

	cached := make(map[string]store.IPGeoCacheEntry)
	if entries, err := p.repo.GetIPGeoCache(unique); err != nil {
		logrus.WithError(err).Warn("读取 IP 归属地缓存失败")
	} else if len(entries) > 0 {
		unknownCached := make([]string, 0)
		for ip, entry := range entries {
			if entry.Domestic == "未知" && entry.Global == "未知" {
				unknownCached = append(unknownCached, ip)
				continue
			}
			cached[ip] = entry
		}
		if len(unknownCached) > 0 {
			if err := p.repo.DeleteIPGeoCache(unknownCached); err != nil {
				logrus.WithError(err).Warn("清理未知 IP 归属地缓存失败")
			}
			enrich.DeleteIPGeoCacheEntries(unknownCached)
		}
		if len(cached) > 0 {
			if err := p.repo.UpdateIPGeoLocations(cached, pendingLocationLabel); err != nil {
				logrus.WithError(err).Warn("回填缓存中的 IP 归属地失败")
			}
		}
	}

	missing := make([]string, 0, len(unique))
	for _, ip := range unique {
		if _, ok := cached[ip]; ok {
			continue
		}
		missing = append(missing, ip)
	}
	if len(missing) == 0 {
		return
	}

	if err := p.repo.UpsertIPGeoPending(missing); err != nil {
		logrus.WithError(err).Warn("写入 IP 归属地待解析队列失败")
	}
}
