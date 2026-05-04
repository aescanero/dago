-- create "graphs" table
CREATE TABLE "graphs" ("id" uuid NOT NULL, "name" character varying NOT NULL, "version" character varying NOT NULL, "description" character varying NULL, "entry_node" character varying NOT NULL, "definition" jsonb NOT NULL, "memory_config" jsonb NULL, "status" character varying NOT NULL DEFAULT 'draft', "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, PRIMARY KEY ("id"));
-- create index "graph_name_version" to table: "graphs"
CREATE UNIQUE INDEX "graph_name_version" ON "graphs" ("name", "version");
-- create index "graph_status" to table: "graphs"
CREATE INDEX "graph_status" ON "graphs" ("status");
-- create "executions" table
CREATE TABLE "executions" ("id" uuid NOT NULL, "status" character varying NOT NULL DEFAULT 'pending', "current_node" character varying NULL, "variables" jsonb NOT NULL, "messages" jsonb NOT NULL, "node_results" jsonb NOT NULL, "error" character varying NULL, "started_at" timestamptz NULL, "completed_at" timestamptz NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "graph_executions" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "executions_graphs_executions" FOREIGN KEY ("graph_executions") REFERENCES "graphs" ("id") ON DELETE NO ACTION);
-- create index "execution_status_graph_executions" to table: "executions"
CREATE INDEX "execution_status_graph_executions" ON "executions" ("status", "graph_executions");
-- create index "execution_status" to table: "executions"
CREATE INDEX "execution_status" ON "executions" ("status");
-- create index "execution_created_at" to table: "executions"
CREATE INDEX "execution_created_at" ON "executions" ("created_at");
-- create "nodes" table
CREATE TABLE "nodes" ("id" uuid NOT NULL, "node_key" character varying NOT NULL, "pattern" character varying NOT NULL, "config" jsonb NOT NULL, "position" jsonb NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "graph_nodes" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "nodes_graphs_nodes" FOREIGN KEY ("graph_nodes") REFERENCES "graphs" ("id") ON DELETE NO ACTION);
-- create index "node_node_key_graph_nodes" to table: "nodes"
CREATE UNIQUE INDEX "node_node_key_graph_nodes" ON "nodes" ("node_key", "graph_nodes");
-- create index "node_pattern_graph_nodes" to table: "nodes"
CREATE INDEX "node_pattern_graph_nodes" ON "nodes" ("pattern", "graph_nodes");
