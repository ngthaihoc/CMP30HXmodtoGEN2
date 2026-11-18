package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hxcore "40hxcore"
)

// runCMP30HXBatchUnlock is the entry point for the batch setup and its startup task.
func runCMP30HXBatchUnlock() error {
	if !hxcore.IsAdmin() {
		return fmt.Errorf("CMP 30HX batch unlock requires an elevated process")
	}

	InitEmbeddedDriverHook()
	var status hxcore.StatusContract
	sessionErr := hxcore.RunTransientDriverSession(true, func(bus hxcore.HardwareBus) error {
		verdict, err := hxcore.NewLinkNegotiator(bus).Unlock(hxcore.NegotiationOptions{
			TargetGen: 2, DeviceID: 0x2189, SkipDisplayRestart: true,
		})
		status = verdict.StatusContract
		return err
	})
	if sessionErr != nil {
		if status.StatusCode == "" || status.StatusCode == hxcore.StatusGen2Success {
			status.StatusCode = hxcore.StatusDrvFail
		}
		status.ErrorCode = "UNLOCK_SESSION_FAILED"
		status.Details = append(status.Details, sessionErr.Error())
	}
	dir := filepath.Dir(hxcore.Gen2StatusPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// cmd.exe's findstr /x requires CRLF line endings.
	report := strings.ReplaceAll(hxcore.FormatStatus(status), "\n", "\r\n")
	if err := os.WriteFile(filepath.Join(dir, "cmp30hx_batch_status.txt"), []byte(report), 0o644); err != nil {
		return err
	}
	if sessionErr != nil {
		return sessionErr
	}
	if status.StatusCode != hxcore.StatusGen2Success {
		return fmt.Errorf("CMP 30HX batch unlock: %s", status.StatusCode)
	}
	return nil
}
