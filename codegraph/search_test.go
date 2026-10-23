package codegraph

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var searchModule = map[string]string{
	"go.mod": "module example.com/s\n\ngo 1.22\n",
	"s.go": `package s

import "fmt"

// Subscribe opts a customer in to the campaign
// and sends the welcome
// message.
func Subscribe(phone string) error { return nil }

func sendWelcomeMessage() {}

type SmsSubscriptionService struct{}

func NewSmsSubscriptionService() *SmsSubscriptionService { return nil }

// Optout removes a subscription.
func (s *SmsSubscriptionService) Optout(id string) {}

func wrap() error { return fmt.Errorf("x") }
`,
}

func hitNames(hs []SearchHit) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.Node.Name
	}
	return out
}

func TestSearch_NameMatchesSubstringsAnywhereCaseInsensitively(t *testing.T) {
	s := buildModule(t, searchModule)

	hits, err := s.Search("SUBSCRIPTION", SearchOptions{Name: true})
	require.NoError(t, err)
	// the struct, the constructor (substring in the middle) and the method (qualified name)
	require.ElementsMatch(t, []string{"SmsSubscriptionService", "NewSmsSubscriptionService", "SmsSubscriptionService.Optout"}, hitNames(hits))
	for _, h := range hits {
		require.Equal(t, MatchName, h.Match)
	}
}

func TestSearch_RanksExactThenPrefixThenSubstringThenDocThenSignature(t *testing.T) {
	s := buildModule(t, searchModule)

	hits, err := s.Search("subscribe", SearchOptions{})
	require.NoError(t, err)
	require.Equal(t, "Subscribe", hits[0].Node.Name)
	require.Equal(t, 0, hits[0].Rank, "exact name")

	welcome, err := s.Search("welcome", SearchOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"sendWelcomeMessage", "Subscribe"}, hitNames(welcome))
	require.Equal(t, []int{2, 3}, []int{welcome[0].Rank, welcome[1].Rank}, "name substring before doc match")
}

func TestSearch_DocPhraseMatchesAcrossLineBreaksAndReturnsSnippet(t *testing.T) {
	s := buildModule(t, searchModule)

	hits, err := s.Search("welcome message", SearchOptions{Doc: true})
	require.NoError(t, err)
	require.Len(t, hits, 1)
	require.Equal(t, "Subscribe", hits[0].Node.Name)
	require.Equal(t, MatchDoc, hits[0].Match)
	require.Contains(t, hits[0].Snippet, "welcome message", "doc lines are joined before matching")
	require.True(t, strings.HasSuffix(hits[0].Snippet, "welcome message."), "match is near the end of a short doc, so no trailing ellipsis: %q", hits[0].Snippet)
}

func TestSearch_SignatureSearchFindsTypesInSignatures(t *testing.T) {
	s := buildModule(t, searchModule)

	// the receiver of a method is part of its signature too, so match on the result type
	hits, err := s.Search(") *SmsSubscriptionService", SearchOptions{Sig: true})
	require.NoError(t, err)
	require.Equal(t, []string{"NewSmsSubscriptionService"}, hitNames(hits), "only the constructor returns the pointer type")

	both, err := s.Search("*SmsSubscriptionService", SearchOptions{Sig: true})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"NewSmsSubscriptionService", "SmsSubscriptionService.Optout"}, hitNames(both), "the pointer receiver is in the signature")
	require.Equal(t, MatchSig, hits[0].Match)
	require.Contains(t, hits[0].Snippet, "func NewSmsSubscriptionService()")
}

func TestSearch_KindFilterAndExternalStubs(t *testing.T) {
	s := buildModule(t, searchModule)

	funcs, err := s.Search("subscri", SearchOptions{Kind: KindFunc})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Subscribe", "NewSmsSubscriptionService"}, hitNames(funcs), "no struct, no method")

	withExt, err := s.Search("errorf", SearchOptions{Name: true})
	require.NoError(t, err)
	require.Equal(t, []string{"Errorf"}, hitNames(withExt))
	require.True(t, withExt[0].Node.Ext)

	hidden, err := s.Search("errorf", SearchOptions{Name: true, HideExternal: true})
	require.NoError(t, err)
	require.Empty(t, hidden)
}

func TestSearch_EmptyQueryIsAnError(t *testing.T) {
	s := buildModule(t, searchModule)
	_, err := s.Search("   ", SearchOptions{})
	require.Error(t, err)
}

func TestSearch_TiesPreferHandWrittenBehaviourOverGeneratedFields(t *testing.T) {
	s := buildModule(t, map[string]string{
		"go.mod":        "module example.com/o\n\ngo 1.22\n",
		"a.go":          "package o\n\ntype Model struct {\n\t// Retry budget for the widget.\n\tRetries int\n}\n\n// Widget retry budget handling.\nfunc Retry() {}\n\n// Widget retry budget helper on the model.\nfunc (m Model) Budget() {}\n",
		"models_gen.go": "package o\n\n// Widget retry budget in generated code.\nfunc GeneratedRetry() {}\n",
	})
	hits, err := s.Search("retry budget", SearchOptions{Doc: true})
	require.NoError(t, err)
	// all four match on doc (same rank): hand-written first, then functions before methods before fields,
	// and generated files last
	require.Equal(t, []string{"Retry", "Model.Budget", "Model.Retries", "GeneratedRetry"}, hitNames(hits))
}

func TestSnippetAround_CutsOnRuneBoundariesAndSurvivesMisalignedOffsets(t *testing.T) {
	text := strings.Repeat("é", 100) + "needle" + strings.Repeat("ü", 100)
	i := strings.Index(text, "needle")
	got := snippetAround(text, i, len("needle"))
	require.Contains(t, got, "needle")
	require.True(t, strings.HasPrefix(got, "…") && strings.HasSuffix(got, "…"))

	require.NotPanics(t, func() { _ = snippetAround("short", 500, 3) })
	require.NotPanics(t, func() { _ = snippetAround("short", -5, 3) })
}
