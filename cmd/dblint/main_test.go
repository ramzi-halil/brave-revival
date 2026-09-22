package main

import (
	"testing"

	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

func TestLintMaster(t *testing.T) {
	master := &pmaster.All{
		Achievement: []*pmaster.Achievement{
			{Id: 1, TextName: 100},
			{Id: 2, TextName: 0},
		},
		TextLang: []*pmaster.TextLang{{Id: 100, Text: "Achievement"}},
	}

	require.NoError(t, lintMaster(master))
}

func TestLintMasterMissingReference(t *testing.T) {
	master := &pmaster.All{
		Achievement: []*pmaster.Achievement{
			{Id: 1, TextName: 200},
			{Id: 2, TextName: 200},
		},
		TextLang: []*pmaster.TextLang{{Id: 100, Text: "Achievement"}},
	}

	err := lintMaster(master)
	require.EqualError(t, err, "found 1 missing foreign key value(s):\n"+
		"master/achievement.text_name references missing id 200 in master/text_lang (2 row(s), first at row 1)")
}
