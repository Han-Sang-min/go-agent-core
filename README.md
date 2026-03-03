# go-agent — Golang 기반 모니터링 Agent/Collector 시스템

Linux / Container / Kubernetes 환경에서 동작하는 **시스템 메트릭 수집 Agent**와
이를 수신하는 **gRPC Collector 서버**, 그리고 운영 시나리오를 검증하는 **Simulator**로 구성된다.

---

## 아키텍처

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                   │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │  Agent   │  │  Agent   │  │  Agent   │  (DaemonSet)  │
│  │ (node-1) │  │ (node-2) │  │ (node-3) │               │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘               │
│       │             │             │                     │
│       │     gRPC (Register/Heartbeat/Metrics)           │
│       │             │             │                     │
│       └─────────────┼─────────────┘                     │
│                     ▼                                   │
│              ┌──────────────┐                           │
│              │  Collector   │  (Deployment)             │
│              │  :50051/tcp  │                           │
│              └──────────────┘                           │
└─────────────────────────────────────────────────────────┘

Agent 수집 경로:
  Host  → /proc/stat, /proc/meminfo, syscall.Statfs
  Container → cgroup v2 (memory.current, cpu.stat)
  K8s   → in-cluster client-go (Pod/Node metadata)
```

### 핵심 흐름

1. **Agent** 가 환경(Host / Container)을 자동 감지하고 적합한 Collector로 메트릭 수집
2. 수집된 CPU, Memory, Disk, Process 메트릭을 **gRPC**로 Collector에 전송
3. Collector는 Agent 등록, Heartbeat 관리, Command 디스패치를 수행
4. Kubernetes 환경에서는 Pod/Node 메타데이터를 추가 수집

---

## 디렉토리 구조

```
go-agent/
├── cmd/
│   ├── agent/main.go          # Agent 엔트리포인트
│   ├── collector/main.go      # Collector 서버 엔트리포인트
│   └── simulator/main.go      # Simulator 엔트리포인트
├── internal/
│   ├── agent/                  # 메트릭 수집, 환경 감지, K8s 연동
│   ├── collector/              # gRPC 서버, 핸들러, Agent 관리
│   ├── config/                 # JSON 설정 로드
│   ├── simulator/              # 다중 Agent 시뮬레이션 & 시나리오
│   └── transport/              # gRPC 클라이언트 래퍼
│   └── util/                   # 유틸리티 함수
├── proto/
│   └── agent.proto             # gRPC 서비스 & 메시지 정의
├── deploy/
│   ├── docker/                 # Dockerfile (agent, collector)
│   └── *.yaml                  # K8s 매니페스트 (namespace, RBAC, DaemonSet, Collector)
├── Makefile
├── config.json
└── README.md
```

---

## 실행 방법

### 사전 요구사항

- Go 1.24+
- protoc + protoc-gen-go, protoc-gen-go-grpc (proto 코드 재생성 시)
- Docker (컨테이너 빌드 시)
- kind / kubectl (K8s 배포 시)

### Host 실행

```bash
# Collector 실행
make run-collector

# Agent 실행 (다른 터미널)
COLLECTOR_ADDR=localhost:50051 make run

# 단일 실행 (1회 수집 후 종료)
make once
```

### Docker 실행

```bash
# 이미지 빌드
docker build -f deploy/docker/Dockerfile.collector -t collector:latest .
docker build -f deploy/docker/Dockerfile.agent -t agent:latest .

# Collector 실행
docker run -p 50051:50051 collector:latest

# Agent 실행
docker run --rm -e COLLECTOR_ADDR=host.docker.internal:50051 agent:latest
```

### Kubernetes 배포

```bash
# Kind 클러스터 생성
kind create cluster --config kind-multi-node.yaml

# 이미지 빌드 & 로드
docker build -f deploy/docker/Dockerfile.agent -t agent:latest .
docker build -f deploy/docker/Dockerfile.collector -t collector:latest .
kind load docker-image agent:latest collector:latest

# 매니페스트 적용
kubectl apply -f deploy/00-namespace.yaml
kubectl apply -f deploy/10-rbac.yaml
kubectl apply -f deploy/30-collector.yaml
kubectl apply -f deploy/20-daemonset.yaml

# 확인
kubectl -n agent-system get pods
```

### Simulator 실행

```bash
# 기본 실행 (5개 Agent, full 시나리오, 30초)
make run-simulator

# 커스텀 실행
bin/simulator -agents=10 -scenario=cpu-spike -duration=60s -interval=500ms

# 시나리오 목록 확인
bin/simulator -list-scenarios
```

**사용 가능 시나리오**: `full`, `cpu-spike`, `mem-spike`, `network-fail`, `error-inject`, `stress`

---

## 설계 선택 이유

### 1. `RuntimeEnv` 인터페이스 기반 환경 추상화

```go
type RuntimeEnv interface {
    Kind() string
    CPU(ctx) CPUStats
    Mem(ctx) MemStats
    Disk(ctx) DiskStats
    Procs(ctx) ProcStats
}
```

Host, Container, Simulated 환경을 동일한 인터페이스로 다루어
**수집 로직과 환경 로직을 분리**했다. Simulator에서도 동일한 `Collect()` 함수를 재사용할 수 있다.

### 2. `/proc`, `/sys` 직접 접근

외부 라이브러리(gopsutil 등)에 의존하지 않고 `/proc/stat`, `/proc/meminfo`, cgroup v2 파일을 직접 파싱한다.
시스템 레벨 이해를 우선하고, 의존성을 최소화했다.

### 3. CPU 사용률 — 차분 계산 (Differential Sampling)

`/proc/stat`의 누적 jiffies를 두 번 샘플링하여 차분으로 CPU 사용률을 계산한다.
순간값이 아닌 **구간 평균**을 구해 더 정확한 수치를 얻는다.

### 4. gRPC 4개 RPC 분리

| RPC | 용도 |
|-----|------|
| `Register` | Agent 등록, UUID 발급 |
| `SendHeartbeat` | 생존 확인 + 펜딩 명령 수신 |
| `SendMetrics` | 메트릭 배치 전송 |
| `ReportCommandResult` | 명령 실행 결과 보고 |

Heartbeat와 Metrics를 분리하여 **제어 채널과 데이터 채널**을 독립시켰다.
Collector가 Heartbeat 응답에 Command를 포함시켜 Agent에 명령을 내릴 수 있다.

### 5. K8s 메타데이터 캐싱 (2분 TTL)

Pod/Node 정보는 자주 변경되지 않으므로 캐싱하여 API Server 부하를 줄였다.
in-cluster config를 사용하고, RBAC은 **최소 권한**(pod get, node get)만 부여한다.

### 6. Collector Agent GC

60초간 Heartbeat가 없는 Agent를 자동 제거하여 메모리 누수를 방지한다.

---

## 한계점 및 개선 방향

### 현재 한계점

- **단일 Collector**: Collector가 단일 인스턴스로, HA 구성이 없다
- **메트릭 저장소 없음**: 수신된 메트릭을 로깅만 하고 영구 저장하지 않는다
- **TLS 미적용**: gRPC 통신이 평문으로 이루어진다
- **Config Hot Reload 미구현**: 설정 변경 시 Agent 재시작이 필요하다
- **cgroup v1 미지원**: cgroup v2만 지원하며, v1 환경에서는 Host 모드로 폴백된다

### 개선 방향

- **Collector HA**: 다수 Collector 인스턴스 + 로드밸런싱
- **메트릭 파이프라인**: 수신 메트릭을 시계열 DB(Prometheus 등)에 저장
- **mTLS**: Agent-Collector 간 상호 인증
- **SIGHUP Hot Reload**: 설정 파일 변경 시 Agent 무중단 반영
- **Plugin 구조**: Collector를 Config 기반으로 동적 등록/해제
- **네트워크 재연결 개선**: 지수 백오프 + 재등록 자동화
