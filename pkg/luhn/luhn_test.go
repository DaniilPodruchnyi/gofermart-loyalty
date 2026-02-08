package luhn

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "valid number 1",
			number: "12345678903",
			want:   true,
		},
		{
			name:   "valid number 2",
			number: "4561261212345467",
			want:   true,
		},
		{
			name:   "valid number 3",
			number: "79927398713",
			want:   true,
		},
		{
			name:   "invalid number",
			number: "1234567890",
			want:   false,
		},
		{
			name:   "empty string",
			number: "",
			want:   false,
		},
		{
			name:   "non-numeric",
			number: "abcd1234",
			want:   false,
		},
		{
			name:   "single digit valid",
			number: "0",
			want:   true,
		},
		{
			name:   "single digit invalid",
			number: "5",
			want:   false,
		},
		{
			name:   "spaces in number",
			number: "1234 5678 903",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.number); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
