package services

import "sync/atomic"

// InputImportActive pauses background reconciliation while a full input reset runs.
var InputImportActive atomic.Bool
