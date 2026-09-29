# Nādabrahma 🎵

> *nada-brahma*, "sound is the divine." A command-line finder for Carnatic **gems**:
> the specific renditions that are hard to stop listening to.

<img width="1770" height="855" alt="image" src="https://github.com/user-attachments/assets/9fc62df3-ec4e-4189-8c4f-7bf6ff3c91c4" />

Most tools point you at the *famous* version of a kriti. Nādabrahma does something
harder. It hunts the particular **(kriti × artist × recording)** where a rendition
transcends every other version, the way one live concert can eclipse a singer's own
studio take. It reads what the crowd actually *feels*, not just how many clicked.

For every kriti it shows you two things:

- **◆ The known version**: the canonical, most-heard recording, as a reference.
- **✦ The gem we found**: the lesser-known rendition our scoring surfaced, with
  what the kriti means, why this recording, and where the magic moment is.

It is free to run. **yt-dlp** does all the YouTube work (no per-video cost, no credit
card, no YouTube API), and **Groq** (free tier) reads comments and writes the framing.

## How it finds gems

A gem is not an artist or a kriti. It is a *rendition*. The same singer can be
transcendent on one kriti and ordinary on another. For each candidate we compute:

| Signal | What it captures |
|---|---|
| **emotion** | Groq reads ~30 real comments for transcendence markers ("goosebumps", "I cry every time", "which sabha? which year?"). The strongest signal. |
| **passion** | comment density: how many people were *compelled to speak*, not raw popularity. |
| **engagement** | like ratio, quality that is harder to game than views. |
| **heat** | YouTube's "most replayed" graph, which also tells us *where* the magic is. |
| **elaboration** | duration, room for *manodharma* (alapana, neraval, swaram). |
| **hidden bonus** | lifts under-exposed renditions so buried gems can win. |

```
gem = 0.30*emotion + 0.24*passion + 0.18*engagement + 0.16*heat + 0.12*elaboration
      * live_bonus * hidden_bonus
```

The candidate pool is gathered from several search angles (studio, live, raga),
screened on free metadata (not views), and only the strongest few get the expensive
comment and emotion pass. That is how a beloved 10K-view recording can beat a
lukewarm 2M-view one.

AI reads meaning, the crowd decides truth. Groq never names recordings (LLMs
hallucinate them). It interprets the kriti and the comments, while real-world signal
ranks the actual videos.

## Install

Requires [Go](https://go.dev) 1.22+, [yt-dlp](https://github.com/yt-dlp/yt-dlp),
and a free [Groq](https://console.groq.com) API key.

```bash
git clone https://github.com/Gajasurve/nadhaBrahma.git
cd nadhaBrahma

pip install -U yt-dlp          # if you don't have it
cp .env.example .env           # then paste your GROQ_API_KEY

go build -o nadabrahma .
./nadabrahma check             # verify yt-dlp, Groq key/model, cache
```

## Usage

```bash
./nadabrahma discover               # deity-shuffle gem feed (the main experience)
./nadabrahma discover Shiva         # pin the feed to one deity
./nadabrahma discover Rama deep     # "deep cuts", only lesser-known renditions
```

Deities: Ganesha, Shiva, Devi, Rama, Krishna, Vishnu, Subrahmanya.

In the feed: `enter` next gem, `o` open in browser, `w` why (score breakdown), `q` quit.

## Configuration

All optional, sensible defaults are built in. See `.env.example`.

| Variable | Default | Meaning |
|---|---|---|
| `GROQ_API_KEY` | (required) | free tier, no card |
| `GROQ_MODEL` | `openai/gpt-oss-120b` | any chat model your key can access |
| `MAX_VIEWS` | `0` (off) | skip renditions above N views (`deep` sets 150000) |
| `COMMENT_DEPTH` | `4` | how many renditions get the AI pass per kriti |
| `MAX_COMMENTS` | `30` | comments read per video |
| `DB_PATH` | `data/gems.db` | local SQLite gem cache |

## Cost

- **yt-dlp is free and unmetered.** Zero cost per video, comments included.
- Only Groq tokens are used, per kriti, on the free tier that is zero. Even at paid
  rates a full kriti is roughly one rupee's fraction.
- Every scored recording is cached locally (`data/gems.db`) and never re-fetched, so
  the corpus compounds and gets cheaper the more you use it.

Scope: Telugu and Sanskrit repertoire (Tyagaraja, Dikshitar, Syama Sastri, Annamacharya).

## Roadmap

- `find "<kriti>"`: best rendition of one kriti (comparative)
- `artist "<name>"`: an artist's personal peaks
- rasikas.org enrichment (grow the seed, mine "best rendition" debates)
- `--analyze`: local audio analysis for a precise fast, slow, or medium tempo label

## License

MIT
