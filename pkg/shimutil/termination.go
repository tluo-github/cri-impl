package shimutil

import (
	"encoding/json"
	"errors"
	"k8s.io/klog"
	"time"
)

const (
	reasonExited   string = "exited"
	reasonSignoled string = "signaled"
)

type TerminationStatus struct {
	raw attrs
}

type attrs struct {
	At       time.Time `json:"at"`
	ExitCode int32     `json:"exitCode"`
	Signal   int32     `json:"signal"`
	Reason   string    `json:"reason"`
}

func ParseExitFile(bytes []byte) (*TerminationStatus, error) {
	raw := attrs{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return nil, err
	}
	if raw.Reason != reasonExited && raw.Reason != reasonSignoled {
		// 意外原因
		return nil, errors.New("Unexpected termination reason")
	}
	if raw.Reason == reasonExited && (raw.ExitCode < 0 || raw.ExitCode > 127) {
		// 意外 exit code
		return nil, errors.New("Unexpected exit code")
	}
	if raw.Reason == reasonSignoled && raw.Signal <= 0 {
		// 意外 signal
		return nil, errors.New("Unexpected signal")
	}
	return &TerminationStatus{raw}, nil
}

func (t *TerminationStatus) IsSignaled() bool {
	return t.raw.Reason == reasonSignoled
}

func (t *TerminationStatus) At() time.Time {
	return t.raw.At
}

func (t *TerminationStatus) ExitCode() int32 {
	if t.IsSignaled() {
		klog.Errorf("ExitCode() should not be used when container terminated has been killed")
		return -1
	}
	return t.raw.ExitCode
}

func (t *TerminationStatus) Signal() int32 {
	if !t.IsSignaled() {
		klog.Errorf("Signal() should not be used when container exited normally")

		return -1
	}
	return t.raw.Signal
}
