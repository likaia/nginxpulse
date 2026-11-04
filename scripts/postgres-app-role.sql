-- Run only as the embedded cluster administrator, before starting the backend.
-- psql variables use identifier/literal quoting; never interpolate env into SQL.
SELECT format('CREATE ROLE %I LOGIN', :'app_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user')
\gexec

BEGIN;
ALTER ROLE :"app_user" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'app_password';

-- Strip inherited privileges too (e.g. pg_read_server_files or SET ROLE admin).
SELECT format('REVOKE %I FROM %I', parent.rolname, :'app_user')
FROM pg_auth_members membership
JOIN pg_roles parent ON parent.oid = membership.roleid
JOIN pg_roles member ON member.oid = membership.member
WHERE member.rolname = :'app_user'
\gexec
COMMIT;

SELECT format('CREATE DATABASE %I OWNER %I', :'app_db', :'app_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'app_db')
\gexec

-- Table/sequence ownership is migrated separately by postgres-app-owner.sql.
-- Database ownership permits DDL without cluster-wide administrative rights.
ALTER DATABASE :"app_db" OWNER TO :"app_user";
