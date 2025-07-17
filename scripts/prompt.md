```
Run the e2e tests with following steps
Build the alert engine
./local_e2e/setup/teardown_local_e2e.sh
./local_e2e/setup/setup_local_e2e.sh
source local_e2e/setup/.env && export SLACK_WEBHOOK_URL && CONFIG_PATH=./local_e2e/setup/config_local_e2e.yaml go run cmd/server/main.go
./run_e2e_tests.sh
```