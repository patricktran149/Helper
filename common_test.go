package helper

import (
	"fmt"
	"testing"
)

func TestGenerateOTP(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "001", args:args{n: 6} , want: 6},
		{name: "002", args:args{n: 6} , want: 6},
		{name: "003", args:args{n: 6} , want: 6},
		{name: "004", args:args{n: 5} , want: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateOTP(tt.args.n)
			if len(got) != tt.want {
				t.Errorf("GenerateOTP() = %v, want %v", len(got), tt.want)
			}

			fmt.Println(got)
		})
	}
}
