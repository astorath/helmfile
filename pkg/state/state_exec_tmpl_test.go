package state

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/roboll/helmfile/pkg/environment"
)

func boolPtrToString(ptr *bool) string {
	if ptr == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&%t", *ptr)
}

func ptr(v interface{}) interface{} {
	r := v
	return reflect.ValueOf(r).Addr().Interface()
}

func TestHelmState_executeTemplates(t *testing.T) {
	tests := []struct {
		name  string
		input ReleaseSpec
		want  ReleaseSpec
	}{
		{
			name: "Has template expressions in chart, values, secrets, set",
			input: ReleaseSpec{
				Chart:     "test-charts/{{ .Release.Name }}",
				Version:   "{{ .Release.Name }}-0.1",
				Verify:    nil,
				Name:      "test-app",
				Namespace: "test-namespace-{{ .Release.Name }}",
				Values:    []interface{}{"config/{{ .Environment.Name }}/{{ .Release.Name }}/values.yaml"},
				Secrets:   []string{"config/{{ .Environment.Name }}/{{ .Release.Name }}/secrets.yaml"},
			},
			want: ReleaseSpec{
				Chart:     "test-charts/test-app",
				Version:   "test-app-0.1",
				Verify:    nil,
				Name:      "test-app",
				Namespace: "test-namespace-test-app",
				Values:    []interface{}{"config/test_env/test-app/values.yaml"},
				Secrets:   []string{"config/test_env/test-app/secrets.yaml"},
			},
		},
		{
			name: "Has template expressions in name and id with recursive refs",
			input: ReleaseSpec{
				Id:        "{{ .Release.Chart }}",
				Chart:     "test-chart",
				Verify:    nil,
				Name:      "{{ .Release.Id }}-{{ .Release.Namespace }}",
				Namespace: "dev",
			},
			want: ReleaseSpec{
				Id:        "test-chart",
				Chart:     "test-chart",
				Verify:    nil,
				Name:      "test-chart-dev",
				Namespace: "dev",
			},
		},
		{
			name: "Has template expressions in boolean values",
			input: ReleaseSpec{
				Id:                 "app",
				Chart:              "test-chart",
				Name:               "app-dev",
				Namespace:          "dev",
				InstalledTemplate:  func(i string) *string { return &i }(`{{ eq .Release.Id "app" | ternary "yes" "no" }}`),
				VerifyTemplate:     func(i string) *string { return &i }(`{{ true }}`),
				Verify:             func(i bool) *bool { return &i }(false),
				WaitTemplate:       func(i string) *string { return &i }(`{{ false }}`),
				TillerlessTemplate: func(i string) *string { return &i }(`yes`),
			},
			want: ReleaseSpec{
				Id:         "app",
				Chart:      "test-chart",
				Name:       "app-dev",
				Namespace:  "dev",
				Installed:  func(i bool) *bool { return &i }(true),
				Verify:     func(i bool) *bool { return &i }(true),
				Wait:       func(i bool) *bool { return &i }(false),
				Tillerless: func(i bool) *bool { return &i }(true),
			},
		},
		// TODO: make complex trees work (values and set values)
		// {
		// 	name: "Has template in values and set-values",
		// 	input: ReleaseSpec{
		// 		Id:        "app",
		// 		Chart:     "test-charts/chart",
		// 		Verify:    nil,
		// 		Name:      "app",
		// 		Namespace: "dev",
		// 		Values:    []interface{}{map[string]string{"key": "{{ .Release.Name }}-val0"}},
		// 		SetValues: []SetValue{
		// 			SetValue{Name: "val1", Value: "{{ .Release.Name }}-val1"},
		// 			SetValue{Name: "val2", File: "{{ .Release.Name }}.yml"},
		// 			SetValue{Name: "val3", Values: []string{"{{ .Release.Name }}-val2", "{{ .Release.Name }}-val3"}},
		// 		},
		// 	},
		// 	want: ReleaseSpec{
		// 		Id:        "app",
		// 		Chart:     "test-charts/chart",
		// 		Verify:    nil,
		// 		Name:      "app",
		// 		Namespace: "dev",
		// 		Values:    []interface{}{map[string]string{"key": "app-val0"}},
		// 		SetValues: []SetValue{
		// 			SetValue{Name: "val1", Value: "test-app-val1"},
		// 			SetValue{Name: "val2", File: "test-app.yml"},
		// 			SetValue{Name: "val3", Values: []string{"test-app-val2", "test-app-val3"}},
		// 		},
		// 	},
		// },
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			state := &HelmState{
				basePath: ".",
				HelmDefaults: HelmSpec{
					KubeContext: "test_context",
				},
				Env:          environment.Environment{Name: "test_env"},
				Namespace:    "test-namespace_",
				Repositories: nil,
				Releases: []ReleaseSpec{
					tt.input,
				},
			}

			r, err := state.ExecuteTemplates()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				t.FailNow()
			}

			actual := r.Releases[0]

			if !reflect.DeepEqual(actual.Id, tt.want.Id) {
				t.Errorf("expected Id %+v, got %+v", tt.want.Id, actual.Id)
			}
			if !reflect.DeepEqual(actual.Name, tt.want.Name) {
				t.Errorf("expected Name %+v, got %+v", tt.want.Name, actual.Name)
			}
			if !reflect.DeepEqual(actual.Chart, tt.want.Chart) {
				t.Errorf("expected Chart %+v, got %+v", tt.want.Chart, actual.Chart)
			}
			if !reflect.DeepEqual(actual.Namespace, tt.want.Namespace) {
				t.Errorf("expected Namespace %+v, got %+v", tt.want.Namespace, actual.Namespace)
			}
			if !reflect.DeepEqual(actual.Values, tt.want.Values) && len(actual.Values) > 0 {
				t.Errorf("expected Values %+v, got %+v", tt.want.Values, actual.Values)
			}
			if !reflect.DeepEqual(actual.Secrets, tt.want.Secrets) && len(actual.Secrets) > 0 {
				t.Errorf("expected Secrets %+v, got %+v", tt.want.Secrets, actual.Secrets)
			}
			if !reflect.DeepEqual(actual.SetValues, tt.want.SetValues) && len(actual.SetValues) > 0 {
				t.Errorf("expected SetValues %+v, got %+v", tt.want.SetValues, actual.SetValues)
			}
			if !reflect.DeepEqual(actual.Version, tt.want.Version) {
				t.Errorf("expected Version %+v, got %+v", tt.want.Version, actual.Version)
			}
			if !reflect.DeepEqual(actual.Installed, tt.want.Installed) {
				t.Errorf("expected actual.Installed %+v, got %+v",
					boolPtrToString(tt.want.Installed), boolPtrToString(actual.Installed),
				)
			}
			if !reflect.DeepEqual(actual.Tillerless, tt.want.Tillerless) {
				t.Errorf("expected actual.Tillerless %+v, got %+v",
					boolPtrToString(tt.want.Tillerless), boolPtrToString(actual.Tillerless),
				)
			}
			if !reflect.DeepEqual(actual.Verify, tt.want.Verify) {
				t.Errorf("expected actual.Verify %+v, got %+v",
					boolPtrToString(tt.want.Verify), boolPtrToString(actual.Verify),
				)
			}
			if !reflect.DeepEqual(actual.Wait, tt.want.Wait) {
				t.Errorf("expected actual.Wait %+v, got %+v",
					boolPtrToString(tt.want.Wait), boolPtrToString(actual.Wait),
				)
			}
		})
	}
}

func TestHelmState_recursiveRefsTemplates(t *testing.T) {

	tests := []struct {
		name  string
		input ReleaseSpec
	}{
		{
			name: "Has reqursive references",
			input: ReleaseSpec{
				Id:        "app-{{ .Release.Name }}",
				Chart:     "test-charts/{{ .Release.Name }}",
				Verify:    nil,
				Name:      "{{ .Release.Id }}",
				Namespace: "dev",
			},
		},
		{
			name: "Has unresolvable boolean templates",
			input: ReleaseSpec{
				Id:           "app",
				Name:         "app-dev",
				Chart:        "test-charts/app",
				Verify:       nil,
				Namespace:    "dev",
				WaitTemplate: func(i string) *string { return &i }("hi"),
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			state := &HelmState{
				basePath: ".",
				HelmDefaults: HelmSpec{
					KubeContext: "test_context",
				},
				Env:          environment.Environment{Name: "test_env"},
				Namespace:    "test-namespace_",
				Repositories: nil,
				Releases: []ReleaseSpec{
					tt.input,
				},
			}

			r, err := state.ExecuteTemplates()
			if err == nil {
				t.Errorf("Expected error, got valid response: %v", r)
				t.FailNow()
			}
		})
	}
}
