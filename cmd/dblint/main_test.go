package main

import (
	"testing"

	"example.com/brave-revival/src/config"
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

	require.NoError(t, lintMaster(master, nil))
}

func TestLintMasterMissingReference(t *testing.T) {
	master := &pmaster.All{
		Achievement: []*pmaster.Achievement{
			{Id: 1, TextName: 200},
			{Id: 2, TextName: 200},
		},
		TextLang: []*pmaster.TextLang{{Id: 100, Text: "Achievement"}},
	}

	err := lintMaster(master, nil)
	require.EqualError(t, err, "found 1 missing foreign key value(s):\n"+
		"master/achievement.text_name references missing id 200 in master/text_lang (2 row(s), first at row 1)")
}

func TestLintResources(t *testing.T) {
	master := &pmaster.All{
		Equipment: []*pmaster.Equipment{
			{Id: 1, IconM: 500},
			{Id: 2, IconM: 0},
		},
	}
	resources := config.Resources{
		"ios": &pmaster.Resources{Resource: map[uint32]*pmaster.ResourceInfo{500: &pmaster.ResourceInfo{}}},
	}

	require.NoError(t, lintMaster(master, resources))
}

func TestLintResourcesMissingReference(t *testing.T) {
	master := &pmaster.All{
		Equipment: []*pmaster.Equipment{
			{Id: 1, IconM: 600},
			{Id: 2, IconM: 600},
		},
	}

	err := lintMaster(master, nil)
	require.EqualError(t, err, "found 1 missing foreign key value(s):\n"+
		"master/equipment.icon_m references missing id 600 in resources (2 row(s), first at row 1)")
}
