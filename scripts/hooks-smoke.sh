#!/usr/bin/env bash
set -euo pipefail

GO_BIN="${GO_BIN:-go}"

run() {
  printf '[hooks-smoke] %s\n' "$*"
  "$@"
}

run "$GO_BIN" test ./internal/hooks -run 'TestCloudflarePurgeURLsDryRunSkipsHTTP|TestGoogleIndexingDryRunSkipsHTTP|TestIndexNowDryRunSkipsHTTP|TestPipelineQueuesPendingJobsWhenHooksDisabled|TestPipelineRunsProvidersWhenHooksEnabled|TestPipelineRecordsRedactedFailureWithoutBreakingSummary|TestHookSummaryMCPUsesSanitizedShape|TestStoreListAndUpdateJobs|TestStoreValidationBranches|TestProvidersRunAndNameInDryRunMode|TestPackagingDocsAndEnvExampleStayNonSecret' -v
run "$GO_BIN" test ./internal/tools -run 'TestMutationToolsAttachHookSummaryAndEnqueueURLs|TestHookAdminToolsAreOptIn|TestHookAdminToolsListRetryStatusAndRunAreSanitized|TestHookWiringHelperBranches' -v
