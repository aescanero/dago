-- reverse: create index "node_pattern_graph_nodes" to table: "nodes"
DROP INDEX "node_pattern_graph_nodes";
-- reverse: create index "node_node_key_graph_nodes" to table: "nodes"
DROP INDEX "node_node_key_graph_nodes";
-- reverse: create "nodes" table
DROP TABLE "nodes";
-- reverse: create index "execution_created_at" to table: "executions"
DROP INDEX "execution_created_at";
-- reverse: create index "execution_status" to table: "executions"
DROP INDEX "execution_status";
-- reverse: create index "execution_status_graph_executions" to table: "executions"
DROP INDEX "execution_status_graph_executions";
-- reverse: create "executions" table
DROP TABLE "executions";
-- reverse: create index "graph_status" to table: "graphs"
DROP INDEX "graph_status";
-- reverse: create index "graph_name_version" to table: "graphs"
DROP INDEX "graph_name_version";
-- reverse: create "graphs" table
DROP TABLE "graphs";
