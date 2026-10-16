package render

// Word lists for the password generators (passwords.go). Hand-picked to be
// safe to hand a competitor: everyday, neutral words -- tech, nature, food,
// objects, places -- nothing crude, nothing that reads as an insult, no
// homophones of either. All lowercase a-z; the generators handle case.
// passwords_test.go enforces the length bounds, charset and uniqueness.

// longWords are 4-7 letters, for passphrase ("Patch-Session-Query-4484").
var longWords = []string{
	"access", "account", "adapter", "address", "admin", "agenda", "alert", "almond", "amber", "anchor",
	"answer", "apple", "apricot", "archive", "arrow", "aspen", "atlas", "audit", "autumn", "avenue",
	"backup", "badge", "bakery", "balance", "bamboo", "banner", "barley", "basket", "beacon", "bench",
	"berry", "bicycle", "binder", "birch", "biscuit", "blanket", "blossom", "bluff", "border", "bottle",
	"branch", "breeze", "bridge", "bright", "brook", "bucket", "buffer", "builder", "bundle", "button",
	"cabin", "cable", "cactus", "cache", "camera", "campus", "candle", "canoe", "canvas", "canyon",
	"carbon", "cargo", "carpet", "castle", "cedar", "cellar", "center", "chalk", "channel", "chapter",
	"cherry", "chimney", "cinder", "circuit", "citrus", "clover", "cluster", "coast", "cobalt", "coconut",
	"coffee", "comet", "command", "compass", "console", "copper", "coral", "cotton", "counter", "cradle",
	"crater", "cricket", "crystal", "cursor", "cypress", "dahlia", "daisy", "debug", "delta", "deploy",
	"desert", "device", "diamond", "digital", "domain", "donut", "drawer", "driver", "dune", "eagle",
	"easel", "ember", "engine", "export", "fabric", "falcon", "feather", "fern", "fiddle", "field",
	"filter", "firefly", "flannel", "flute", "folder", "forest", "fossil", "fennel", "galaxy", "garden",
	"garnet", "gateway", "gazebo", "geyser", "ginger", "glacier", "globe", "granite", "graph", "gravel",
	"grove", "guitar", "hammock", "harbor", "harvest", "hazel", "header", "helmet", "heron", "hickory",
	"hollow", "honey", "horizon", "iceberg", "import", "index", "indigo", "inlet", "island", "ivory",
	"jacket", "jasmine", "journal", "juniper", "kayak", "kernel", "kettle", "keypad", "kitten", "ladder",
	"lagoon", "lantern", "laptop", "lattice", "lemon", "lentil", "letter", "library", "lilac", "linen",
	"locket", "lotus", "lumber", "magnet", "mango", "maple", "marble", "meadow", "melody", "memory",
	"meteor", "mirror", "mitten", "module", "monitor", "mosaic", "muffin", "network", "nickel", "noodle",
	"nutmeg", "oasis", "object", "ocean", "olive", "orange", "orbit", "orchard", "otter", "outlet",
	"packet", "paddle", "palette", "panda", "parcel", "parrot", "pastry", "patch", "pebble", "pelican",
	"pencil", "pepper", "pigeon", "pillow", "pilot", "pine", "planet", "plaza", "pocket", "pollen",
	"poplar", "portal", "potato", "prairie", "prism", "profile", "pumpkin", "puzzle", "quarry", "query",
	"quill", "quilt", "rabbit", "radar", "radish", "rainbow", "raven", "record", "reef", "relay",
	"report", "ribbon", "ridge", "river", "rocket", "router", "saddle", "salmon", "sandal", "satchel",
	"schema", "script", "season", "server", "session", "shadow", "shelter", "signal", "silver", "socket",
	"spruce", "squash", "stable", "stream", "summit", "sunset", "switch", "syntax", "tablet", "teapot",
	"tensor", "thistle", "thunder", "timber", "toast", "token", "topaz", "tower", "tracker", "trail",
	"tulip", "tunnel", "turtle", "update", "valley", "vector", "velvet", "violet", "voyage", "waffle",
	"walnut", "wander", "willow", "window", "winter", "wizard", "yogurt", "zephyr", "zipper", "zinnia",
}

// shortWords are 3-5 letters, for wordchain ("fork-Brick-Spoon-Snail-test").
var shortWords = []string{
	"acorn", "apple", "arch", "atom", "bagel", "bark", "basil", "bead", "bean", "bell",
	"bench", "berry", "bird", "blue", "boat", "bolt", "book", "boot", "bowl", "brick",
	"brush", "bud", "bulb", "cake", "calm", "camp", "cane", "card", "cart", "cave",
	"chair", "chalk", "chip", "clam", "clay", "cliff", "cloud", "coal", "coat", "code",
	"coin", "cone", "cook", "cord", "corn", "crab", "crane", "crow", "cube", "cup",
	"dart", "dawn", "deck", "deer", "desk", "dial", "disk", "dock", "dove", "drum",
	"duck", "dune", "dusk", "echo", "edge", "elm", "fable", "fawn", "fern", "field",
	"fig", "file", "fish", "flag", "flame", "flute", "foam", "fog", "fork", "fox",
	"frog", "frost", "fudge", "gate", "gear", "gem", "glow", "goat", "gold", "grain",
	"grape", "grass", "gull", "harp", "hawk", "hay", "hazel", "heron", "hill", "hive",
	"hook", "horn", "hut", "iris", "iron", "isle", "ivy", "jade", "jam", "jar",
	"jazz", "jet", "kale", "kelp", "kite", "knot", "lake", "lamp", "lane", "leaf",
	"lemon", "lily", "lime", "link", "lion", "loaf", "lock", "loft", "loop", "lynx",
	"map", "maple", "mars", "mask", "melon", "mesa", "mint", "mist", "moon", "moss",
	"moth", "mug", "nest", "net", "node", "nut", "oak", "oar", "oat", "olive",
	"onion", "opal", "orb", "owl", "palm", "pan", "park", "path", "peach", "pear",
	"pearl", "pen", "pine", "pipe", "plum", "pond", "port", "pot", "quail", "quilt",
	"raft", "rain", "reed", "reef", "ring", "road", "robin", "rock", "roof", "rope",
	"rose", "ruby", "rust", "sage", "sail", "salt", "sand", "scarf", "seal", "seed",
	"shell", "ship", "silk", "sky", "sled", "slate", "snail", "snow", "sock", "soup",
	"spade", "spoon", "star", "stem", "stone", "stool", "storm", "sun", "swan", "table",
	"test", "tide", "tile", "toad", "tool", "torch", "tower", "trail", "tree", "tulip",
	"tuna", "twig", "vase", "vine", "wand", "wave", "wheat", "whale", "wind", "wing",
	"wolf", "wood", "wool", "wren", "yak", "yard", "yarn", "zebra", "zinc", "zone",
}
