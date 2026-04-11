package service

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestResetStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		statusCode       int
		statusCodeConfig string
		expectedCode     int
	}{
		{
			name:             "map string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"503"}`,
			expectedCode:     503,
		},
		{
			name:             "map int value",
			statusCode:       429,
			statusCodeConfig: `{"429":503}`,
			expectedCode:     503,
		},
		{
			name:             "skip invalid string value",
			statusCode:       429,
			statusCodeConfig: `{"429":"bad-code"}`,
			expectedCode:     429,
		},
		{
			name:             "skip status code 200",
			statusCode:       200,
			statusCodeConfig: `{"200":503}`,
			expectedCode:     200,
		},
		{
			name:             "fallback 504 to 502 without config",
			statusCode:       504,
			statusCodeConfig: "",
			expectedCode:     502,
		},
		{
			name:             "fallback 524 to 502 without config",
			statusCode:       524,
			statusCodeConfig: "",
			expectedCode:     502,
		},
		{
			name:             "fallback 504 to 502 when config invalid json",
			statusCode:       504,
			statusCodeConfig: `{"504":`,
			expectedCode:     502,
		},
		{
			name:             "respect explicit mapping over fallback",
			statusCode:       504,
			statusCodeConfig: `{"504":599}`,
			expectedCode:     599,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newAPIError := &types.NewAPIError{
				StatusCode: tc.statusCode,
			}
			ResetStatusCode(newAPIError, tc.statusCodeConfig)
			require.Equal(t, tc.expectedCode, newAPIError.StatusCode)
		})
	}
}
