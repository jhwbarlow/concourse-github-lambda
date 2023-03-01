package template

import (
	"strings"
	"text/template"
)

// NewTemplate for github key title and secrets manager path.
func NewTemplate(team, repository, owner, template string) *Template {
	return &Template{
		Team:  team,
		Owner: owner,
		// sanitise the secrets manager path as concourse treats dots as delimiters
		Repository: strings.ReplaceAll(repository, ".", "-"),
		Template:   template,
	}
}

// NewTemplateWithoutRepository creates a template in cases where the possibility of including the repository name in the template does not apply.
func NewTemplateWithoutRepository(team, owner, template string) *Template {
	return &Template{
		Team:       team,
		Owner:      owner,
		Repository: "",
		Template:   template,
	}
}

// Template ...
type Template struct {
	Team       string
	Owner      string
	Repository string
	Template   string
}

func (p *Template) String() (string, error) {
	t, err := template.New("path").Option("missingkey=error").Parse(p.Template)
	if err != nil {
		return "", err
	}
	var s strings.Builder
	if err = t.Execute(&s, p); err != nil {
		return "", err
	}
	return s.String(), nil
}
