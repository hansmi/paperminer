package factertest

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/aurum"
	"github.com/hansmi/dossier"
	"github.com/hansmi/paperminer"
	"go.uber.org/zap/zaptest"
)

func init() {
	aurum.Init()
}

func defaultGolden() aurum.Golden {
	return aurum.Golden{
		Dir:   "./testdata",
		Codec: &aurum.JSONCodec{},
		CmpOptions: cmp.Options{
			cmpopts.EquateEmpty(),
			cmpopts.EquateApproxTime(0),
		},
	}
}

// TestCase implements a generic set of tests for document facters.
type TestCase struct {
	golden      aurum.Golden
	facter      paperminer.DocumentFacter
	testTimeout time.Duration
}

func NewTestCase(t *testing.T, df paperminer.DocumentFacter) *TestCase {
	t.Helper()

	return &TestCase{
		golden:      defaultGolden(),
		facter:      df,
		testTimeout: time.Minute,
	}
}

func (tc *TestCase) makeContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), tc.testTimeout)
	t.Cleanup(cancel)

	return ctx
}

func makeGoldenName(path string) string {
	name := filepath.Base(path)

	return name[:len(name)-len(filepath.Ext(name))] + ".want.json"
}

func stringifyFacts(c aurum.Codec, facts *paperminer.Facts) string {
	if facts != nil {
		if buf, err := c.Marshal(facts); err == nil {
			return string(buf)
		}
	}

	return fmt.Sprintf("%#v", facts)
}

func (tc *TestCase) Assert(t *testing.T, input string) {
	t.Helper()
	t.Run(filepath.Base(input), func(t *testing.T) {
		ctx := tc.makeContext(t)

		doc := dossier.NewDocument(input)

		if err := doc.Validate(ctx); err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		facts, err := tc.facter.DocumentFacts(ctx, paperminer.DocumentFacterOptions{
			Logger:   zaptest.NewLogger(t),
			Document: doc,
		})
		if err != nil {
			t.Fatalf("DocumentFacts() failed: %v", err)
		}

		t.Logf("Discovered facts:\n%s", stringifyFacts(tc.golden.Codec, facts))

		tc.golden.Assert(t, makeGoldenName(input), facts)
	})
}
