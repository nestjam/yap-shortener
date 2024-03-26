package shortener

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShorten(t *testing.T) {
	type args struct {
		id uint32
	}
	tests := []struct {
		name string
		want string
		args args
	}{
		{
			want: "y",
			args: args{
				id: 0,
			},
		},
		{
			want: "n",
			args: args{
				id: 1,
			},
		},
		{
			want: "yn",
			args: args{
				id: alphabetLen,
			},
		},
		{
			want: "yyn",
			args: args{
				id: alphabetLen * alphabetLen,
			},
		},
		{
			want: "zyn",
			args: args{
				id: alphabetLen*alphabetLen + 55,
			},
		},
		{
			name: "max int",
			want: "gf1psJ",
			args: args{
				id: math.MaxInt32,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Shorten(tt.args.id)
			assert.Equal(t, tt.want, got)
		})
	}
}
