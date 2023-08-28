package tests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRespondsWithLove(t *testing.T) {
	resp, err := http.Get(fmt.Sprintf("%s/ping", os.Getenv("host")))
	require.NoError(t, err, "HTTP error")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "HTTP status code")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read HTTP body")

	// Finally, test the business requirement!
	require.Equal(t, "pong", string(body), "Wrong ping response")
}
