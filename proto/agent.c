syntax = "proto3";

package agent.v1;
option go_package = "go-agent/proto/agentv1;agentv1";

import "google/protobuf/timestamp.proto";

service CollectorService {
    rpc SendHeartbeat(Heartbeat) returns (Ack);
    rpc SendMetrics(MetricBatch) returns (Ack)
}

message Heartbeat {
    string agent_id = 1;
    string hostname = 2;
    google.protobuf.Timestamp time = 3;
}

message Metric {
    string name = 1;
    double value = 2;
    string unit = 3;
}

message MetricBatch {
    string agent_id = 1;
    googl.protobuf.Timestamp time = 2;
    repeated Metric metrics = 3;
}

message Ack {
    bool ok = 1;
    string message = 2;
}
