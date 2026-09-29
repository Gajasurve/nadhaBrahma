package main

// SeedKriti is a canonical composition we trust as a discovery starting point.
// Curated by hand from the Telugu/Sanskrit repertoire so v1 needs no scraping.
// The rasikas.org enrichment crawler (later) will grow this corpus.
type SeedKriti struct {
	Title    string
	Composer string
	Deity    string
	Raga     string
	Language string
	Gloss    string // one-line meaning — seeds the hook even if Groq is offline
}

// seedKritis — accurate canon. Scope: Telugu/Sanskrit composers only
// (Tyagaraja, Dikshitar, Syama Sastri, Annamacharya). Tamil/Malayalam/Kannada out.
var seedKritis = []SeedKriti{
	// ── Ganesha ──
	{"Vatapi Ganapatim Bhaje", "Muthuswami Dikshitar", "Ganesha", "Hamsadhwani", "Sanskrit", "Invocation to Ganapati of Vatapi — remover of obstacles, dwelling in the heart."},
	{"Gananayakam", "Muthuswami Dikshitar", "Ganesha", "Rudrapriya", "Sanskrit", "Salutation to the leader of the ganas, the source of auspiciousness."},
	{"Mahaganapatim Manasa Smarami", "Muthuswami Dikshitar", "Ganesha", "Nata", "Sanskrit", "I remember the great Ganapati within my mind."},

	// ── Shiva / Nataraja ──
	{"Ananda Natana Prakasam", "Muthuswami Dikshitar", "Shiva", "Kedaram", "Sanskrit", "The radiance of Shiva's blissful cosmic dance at Chidambaram."},
	{"Sri Dakshinamurtim", "Muthuswami Dikshitar", "Shiva", "", "Sanskrit", "Shiva as Dakshinamurthy — the guru who teaches through silence."},

	// ── Devi ──
	{"Meenakshi Memudam Dehi", "Muthuswami Dikshitar", "Devi", "Purvikalyani", "Sanskrit", "Grant me joy, O Meenakshi, whose glance showers nectar."},
	{"Saroja Dala Netri", "Syama Sastri", "Devi", "Sankarabharanam", "Telugu", "O lotus-eyed Devi, protect me — you are my only refuge."},
	{"Devi Brova Samayamide", "Syama Sastri", "Devi", "Chintamani", "Telugu", "O Devi, this is the very moment to protect me."},
	{"Nannu Brovu Lalita", "Syama Sastri", "Devi", "Lalita", "Telugu", "Protect me, O Lalita — mother of the three worlds."},
	{"Shankari Shankuru Chandramukhi", "Syama Sastri", "Devi", "Saveri", "Sanskrit", "O consort of Shankara, moon-faced bestower of all good."},

	// ── Rama ──
	{"Nagumomu Galavani", "Tyagaraja", "Rama", "Abheri", "Telugu", "The one with the smiling face — why do you hide it from me?"},
	{"Endaro Mahanubhavulu", "Tyagaraja", "Rama", "Sri", "Telugu", "Salutations to the countless great souls — I bow to them all."},
	{"Chakkani Raja Margamu", "Tyagaraja", "Rama", "Kharaharapriya", "Telugu", "When a royal highway of devotion exists, why wander the by-lanes?"},
	{"Mokshamu Galada", "Tyagaraja", "Rama", "Saramati", "Telugu", "Is liberation possible for those without devotion and true music?"},
	{"Nannu Palimpa", "Tyagaraja", "Rama", "Mohanam", "Telugu", "To protect me, Rama himself has come."},

	// ── Krishna ──
	{"Bhavayami Gopalabalam", "Muthuswami Dikshitar", "Krishna", "Yamunakalyani", "Sanskrit", "I meditate on the child Gopala, the cowherd of Brindavan."},
	{"Brahmam Okate", "Annamacharya", "Krishna", "Bauli", "Telugu", "The supreme is one — there is no high or low among beings."},
	{"Nanati Bratuku", "Annamacharya", "Krishna", "Revagupti", "Telugu", "This fleeting life is but a bubble — a passing play of illusion."},

	// ── Vishnu / Venkateswara ──
	{"Adivo Alladivo", "Annamacharya", "Vishnu", "Madhyamavati", "Telugu", "There — that is the abode of Venkateswara upon the seven hills."},
	{"Hiranmayim Lakshmim", "Muthuswami Dikshitar", "Vishnu", "Lalita", "Sanskrit", "The golden Lakshmi — I take refuge, free me from birth and death."},

	// ── Subrahmanya / Murugan ──
	{"Sri Subrahmanyaya Namaste", "Muthuswami Dikshitar", "Subrahmanya", "Kambhoji", "Sanskrit", "Salutations to Subrahmanya, commander of the divine forces."},
}

// deities returns the unique deity list, in a stable order, for rotation.
func deities() []string {
	seen := map[string]bool{}
	var out []string
	for _, k := range seedKritis {
		if !seen[k.Deity] {
			seen[k.Deity] = true
			out = append(out, k.Deity)
		}
	}
	return out
}

// kritisFor returns all seed kritis for a deity.
func kritisFor(deity string) []SeedKriti {
	var out []SeedKriti
	for _, k := range seedKritis {
		if k.Deity == deity {
			out = append(out, k)
		}
	}
	return out
}
