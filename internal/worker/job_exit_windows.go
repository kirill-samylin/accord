//go:build windows

package worker

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Layout from JOBOBJECT_BASIC_ACCOUNTING_INFORMATION (winnt.h).
type jobAccounting struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

func finishWindowsJob(job windows.Handle) (forced bool, err error) {
	deadline := time.Now().Add(3 * time.Second)
	for {
		var info jobAccounting
		if err = windows.QueryInformationJobObject(job, windows.JobObjectBasicAccountingInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)), nil); err != nil {
			return forced, errOwnership
		}
		if info.ActiveProcesses == 0 {
			return forced, nil
		}
		if !forced {
			if err = windows.TerminateJobObject(job, 1); err != nil {
				return true, errOwnership
			}
			forced = true
		}
		if time.Now().After(deadline) {
			return forced, errOwnership
		}
		time.Sleep(20 * time.Millisecond)
	}
}
