import { Link } from "react-router-dom";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { GraphStatusBadge } from "./GraphStatusBadge";
import type { GraphResponse } from "@/api/types.gen";

interface Props {
  graphs: GraphResponse[];
}

export function GraphTable({ graphs }: Props) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Version</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {graphs.map((graph) => (
          <TableRow key={graph.id}>
            <TableCell className="font-medium">
              <Link
                to={`/graphs/${graph.id}`}
                className="hover:underline text-primary"
              >
                {graph.name}
              </Link>
            </TableCell>
            <TableCell>{graph.version}</TableCell>
            <TableCell>
              <GraphStatusBadge status={graph.status} />
            </TableCell>
            <TableCell>
              {new Date(graph.created_at).toLocaleDateString()}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
