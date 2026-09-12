package main

import "testing"

func TestIndependentWingetObservationFailsClosed(t *testing.T) {
	for _, c := range []struct { code int; output string; failed bool; want string }{
		{0,"Name Vendor.App 1.0 winget\n",false,"present"},
		{0,"Name Vendor.App.Helper 1.0 winget\n",false,"unknown"},
		{0,"",false,"unknown"},
		{-1978335212,"No packages found",true,"absent"},
		{-1978335231,"Source failed",true,"unknown"},
		{-1,"Name Vendor.App 1.0 winget\n",true,"unknown"},
	} {
		if got := independentWingetState("Vendor.App",c.code,c.output,c.failed); got != c.want { t.Fatalf("code %d output %q: %s, want %s",c.code,c.output,got,c.want) }
	}
}
