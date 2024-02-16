package invoice

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/hansmi/dossier/pkg/sketch"
	"github.com/hansmi/paperminer"
	"github.com/hansmi/paperminer/internal/ref"
	"github.com/hansmi/paperminer/pkg/sketchfacts"
)

//go:embed sketch.textproto
var sketchTextproto string

var Plugin = sketchfacts.MustNew(sketchfacts.Options{
	Name:      "invoice",
	Textproto: sketchTextproto,
	Required:  []string{"correspondent", "bill_total"},
	Build:     build,
})

func build(report *sketch.PageReport) (*paperminer.Facts, error) {
	facts := &paperminer.Facts{
		Correspondent: ref.Ref("Acme Lawn Care"),
	}

	title := []string{"Invoice"}

	if node := report.NodeByName("bill_total"); node != nil {
		valueText := strings.TrimSpace(node.TextMatch().MustNamed("value").Text)

		value, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return nil, err
		}

		if currencyText := strings.TrimSpace(node.TextMatch().MustNamed("currency").Text); currencyText != "" {
			switch currencyText {
			case "\u20ac":
				currencyText = "Euro"
			}

			title = append(title, currencyText)
		}

		title = append(title, fmt.Sprintf("%.2f", value))
	}

	facts.Title = ref.Ref(strings.Join(title, ", "))

	facts.SetTags = append(facts.SetTags, "invoice")

	return facts, nil
}

func init() {
	paperminer.MustRegisterPlugin(Plugin)
}
