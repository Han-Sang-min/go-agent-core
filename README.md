# Step 01 — Agent Core Bootstrap

본 단계에서는 Golang 기반 Agent(Daemon)의 **기본 실행 골격**을 구현했다.
Linux 환경에서 장시간 실행되는 프로세스를 전제로,
**안전한 종료(graceful shutdown)** 와 **주기적 작업 루프**를 중심으로 설계했다.

## 구현 내용

### 1. Graceful Shutdown
- `SIGINT`, `SIGTERM` 시그널을 수신하여 정상 종료 경로로 진입
- `os.Exit`를 사용하지 않고, `return` 기반 종료로 `defer`가 정상 동작하도록 설계
- ticker, signal notifier 등 리소스를 안전하게 정리

### 2. 주기 실행 루프
- `time.Ticker` 기반으로 Agent의 주기적 작업 구조 구현
- 향후 Metric 수집 로직을 삽입할 수 있도록 worker 함수로 분리

### 3. CLI 옵션 지원
- `flag` 패키지를 사용하여 다음 옵션 지원
  - `-config`: 설정 파일 경로
  - `-once`: 단일 실행 모드
- 실행 모드에 따라 루프 실행 여부를 분기 처리

### 4. 로그 타임스탬프
- 로그 시간은 `RFC3339` 포맷으로 출력
- 컨테이너 / 분산 환경에서 시간 정합성을 고려한 포맷 선택

## 설계 의도

- Agent는 장시간 실행되는 프로세스이므로,
  비정상 종료보다는 **명시적인 종료 경로**를 갖도록 설계
- 초기 단계부터 운영 환경을 가정하여
  이후 Step에서의 확장(Collector, gRPC, K8s)을 고려한 구조 유지
