-- Convert tags columns from text[] to jsonb.
-- Three-step pattern required: DROP DEFAULT first so PostgreSQL does not try
-- to cast the text[] literal '{}' to jsonb (it cannot do it automatically),
-- then ALTER TYPE with explicit USING clause for existing row data,
-- then restore the DEFAULT as a jsonb empty array.

ALTER TABLE "org_units" ALTER COLUMN "tags" DROP DEFAULT;
ALTER TABLE "org_units"
    ALTER COLUMN "tags" TYPE jsonb
    USING array_to_json("tags")::jsonb;
ALTER TABLE "org_units" ALTER COLUMN "tags" SET DEFAULT '[]'::jsonb;

ALTER TABLE "users" ALTER COLUMN "tags" DROP DEFAULT;
ALTER TABLE "users"
    ALTER COLUMN "tags" TYPE jsonb
    USING array_to_json("tags")::jsonb;
ALTER TABLE "users" ALTER COLUMN "tags" SET DEFAULT '[]'::jsonb;
