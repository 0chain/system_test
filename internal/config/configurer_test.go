package config

import (
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"testing"
)

func TestNewConfigurer(t *testing.T) {
	type args struct {
		fileLocation string
	}
	tests := []struct {
		name    string
		args    args
		want    *Configurer
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				fileLocation: "./test_with/system_test.yaml",
			},
			want: &Configurer{
				cfg: &viper.Viper{},
				RequiredConfig: &RequiredConfig{
					DNSHostName: StringToStringPointer("https://dev.0chain.net"),
					LogLevel: StringToStringPointer("error"),
					},
				},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfigurer(tt.args.fileLocation)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfigurer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got.RequiredConfig, tt.want.RequiredConfig); diff != "" {
				t.Errorf("NewConfigurer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func StringToStringPointer(stringToConvert string) *string {
	return &stringToConvert
}
