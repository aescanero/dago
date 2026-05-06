-- create "org_units" table
CREATE TABLE "org_units" ("id" uuid NOT NULL, "name" character varying NOT NULL, "path" character varying NOT NULL, "tags" text[] NOT NULL DEFAULT '{}', "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "org_unit_children" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "org_units_org_units_children" FOREIGN KEY ("org_unit_children") REFERENCES "org_units" ("id") ON DELETE SET NULL);
-- create index "orgunit_path" to table: "org_units"
CREATE UNIQUE INDEX "orgunit_path" ON "org_units" ("path");
-- create "users" table
CREATE TABLE "users" ("id" uuid NOT NULL, "email" character varying NOT NULL, "password_hash" character varying NOT NULL, "tags" text[] NOT NULL DEFAULT '{}', "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "org_unit_users" uuid NULL, PRIMARY KEY ("id"), CONSTRAINT "users_org_units_users" FOREIGN KEY ("org_unit_users") REFERENCES "org_units" ("id") ON DELETE SET NULL);
-- create index "user_email" to table: "users"
CREATE UNIQUE INDEX "user_email" ON "users" ("email");
