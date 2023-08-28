package courier

import (
	"context"
	"testing"
	"tests/suites/postgres"

	"github.com/stretchr/testify/suite"
)

type CourierTestSuite struct {
	suite.Suite
	pgSuite   *postgres.Suite
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (s *CourierTestSuite) SetupSuite() {
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	s.pgSuite = postgres.SetupInstance(s.ctx)
}

func (s *CourierTestSuite) TearDownSuite() {
	s.pgSuite.TearDownInstance()
	s.ctxCancel()
}

func (s *CourierTestSuite) TearDownTest() {
	s.pgSuite.TruncateAll()
}

func TestCourierTestSuite(t *testing.T) {
	suite.Run(t, new(CourierTestSuite))
}
