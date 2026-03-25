// Package words provides word lists for generating memorable subdomain names.
package words

import (
	"fmt"
	"math/rand/v2"
)

var adjectives = []string{
	"amber", "brave", "calm", "dark", "eager", "fair", "glad", "hazy",
	"idle", "jade", "keen", "lazy", "mild", "neat", "odd", "pale",
	"quick", "rare", "sage", "tame", "umber", "vast", "warm", "exact",
	"young", "zeal", "acid", "bold", "cool", "deft", "epic", "fast",
	"grim", "hard", "icy", "just", "kind", "lush", "mad", "nice",
	"open", "pure", "quiet", "rich", "slim", "tall", "used", "vivid",
	"wise", "arid", "blue", "crisp", "dry", "easy", "firm", "gold",
	"huge", "icy", "jolly", "keen", "loud", "merry", "noble", "opal",
	"pink", "real", "soft", "true", "urban", "viral", "wild", "xenial",
	"yellow", "zesty", "agile", "breezy", "clever", "dusty", "elder",
	"foggy", "gruff", "hazy", "indie", "jumpy", "kinky", "lithe",
	"misty", "nervy", "onyx", "plush", "rusty", "silky", "tangy",
	"ultra", "vague", "wavy", "xeric", "zippy", "ample", "balmy",
	"chilly", "damp", "elfin", "funky", "grassy", "hasty", "inky",
	"jazzy", "lofty", "murky", "nimble", "olive", "peppy", "quirky",
	"rainy", "snowy", "tidy", "unruly", "vivid", "wiry", "yummy",
}

var nouns = []string{
	"ape", "bear", "crow", "deer", "elk", "fox", "gnu", "hare",
	"ibis", "jay", "kite", "lynx", "mink", "newt", "owl", "puma",
	"quail", "raven", "seal", "toad", "urial", "vole", "wolf", "xenops",
	"yak", "zebu", "ant", "bee", "cod", "dove", "emu", "frog",
	"gull", "hawk", "imp", "joey", "koi", "lark", "mole", "nene",
	"oryx", "pike", "quoll", "rat", "slug", "tern", "urchin", "viper",
	"wren", "yeti", "zorilla", "asp", "bass", "clam", "dace", "egret",
	"finch", "gecko", "heron", "iris", "jackal", "kudu", "llama",
	"moose", "narwhal", "orca", "parrot", "quetzal", "robin", "swan",
	"tiger", "uakari", "vulture", "walrus", "xerus", "yaffle", "zebra",
	"alga", "bison", "crane", "dhole", "earwig", "falcon", "gorilla",
	"hyena", "iguana", "jaguar", "kakapo", "lemur", "marmot", "numbat",
	"ocelot", "panda", "quahog", "rhino", "skink", "tapir", "umbra",
	"vixen", "wombat", "xiphos", "yapok", "zorilla", "auk", "bongo",
	"civet", "dingo", "ermine", "ferret", "gibbon", "hoopoe", "impala",
}

// Random returns a random two-word subdomain like "brave-llama".
func Random() string {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	return fmt.Sprintf("%s-%s", adj, noun)
}
