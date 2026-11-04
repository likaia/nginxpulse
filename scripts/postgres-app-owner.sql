-- Only transfer application objects inside the application database. REASSIGN
-- OWNED cannot be used for the bootstrap role: it owns required system objects.
-- Re-running this after an interrupted upgrade is safe.
SELECT format('ALTER %s %I.%I OWNER TO %I',
    CASE c.relkind
        WHEN 'r' THEN 'TABLE' WHEN 'p' THEN 'TABLE' WHEN 'S' THEN 'SEQUENCE'
        WHEN 'v' THEN 'VIEW' WHEN 'm' THEN 'MATERIALIZED VIEW' WHEN 'f' THEN 'FOREIGN TABLE'
    END, n.nspname, c.relname, :'app_user')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_roles r ON r.oid = c.relowner
WHERE r.rolname = :'admin_user'
  AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\' AND n.nspname <> 'information_schema'
  AND c.relkind IN ('r', 'p', 'S', 'v', 'm', 'f')
-- Transfer tables first; their owned sequences follow their table ownership.
ORDER BY c.relkind = 'S', c.oid
\gexec

SELECT format('ALTER SCHEMA %I OWNER TO %I', n.nspname, :'app_user')
FROM pg_namespace n JOIN pg_roles r ON r.oid = n.nspowner
WHERE r.rolname = :'admin_user'
  AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\' AND n.nspname <> 'information_schema'
\gexec
