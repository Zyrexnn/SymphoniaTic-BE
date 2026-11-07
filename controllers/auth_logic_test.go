package controllers_test

import (
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/controllers"
)

func TestOTPConstants(t *testing.T) {
	if controllers.OTPCooldownDuration != 60*time.Second {
		t.Errorf("Expected 60s cooldown duration, got: %v", controllers.OTPCooldownDuration)
	}

	if controllers.OTPTTL != 5*time.Minute {
		t.Errorf("Expected 5m OTP TTL, got: %v", controllers.OTPTTL)
	}

	if controllers.MaxOTPAttempts != 5 {
		t.Errorf("Expected 5 max OTP attempts, got: %d", controllers.MaxOTPAttempts)
	}
}
