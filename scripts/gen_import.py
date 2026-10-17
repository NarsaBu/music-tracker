"""Builds a Music Tracker import file (.txt, one artist per line) from a music
library laid out as <library>/<Artist>/<Album>.

Usage: python3 scripts/gen_import.py "/Volumes/Toshiba 4TB/Music" music-artists.txt
"""
import os, sys
src, out = sys.argv[1], sys.argv[2]

# Compilations and cast/soundtrack folders, not artists.
skip = {"Various Artists", "Rock", "Amphibious Zoo Music", "Jesus Christ Superstar"}

# Folders holding a collaboration: subscribe to each participant.
split = {
    "Aikko, INSPACE & katanacss": ["Aikko", "INSPACE", "katanacss"],
    "DJ KRAS & RISHA": ["DJ KRAS", "RISHA"],
    "Dreamville & J. Cole": ["Dreamville", "J. Cole"],
    "Gentlemens Club & SampliFire": ["Gentlemens Club", "SampliFire"],
    "Gruppa Skryptonite & Therr Maitz": ["Gruppa Skryptonite", "Therr Maitz"],
    "Gucci Mane, Bruno Mars & Kodak Black": ["Gucci Mane", "Bruno Mars", "Kodak Black"],
    "hxvvxn & setinai": ["hxvvxn", "setinai"],
    "Jackson Wang & Galantis": ["Jackson Wang", "Galantis"],
    "Jauz & Crankdat": ["Jauz", "Crankdat"],
    "Jean-Michel Blais & CFCF": ["Jean-Michel Blais", "CFCF"],
    "JEEMBO & TVETH": ["JEEMBO", "TVETH"],
    "Jonas Blue & JP Cooper": ["Jonas Blue", "JP Cooper"],
    "Julius Dreisig & Zeus X Crona": ["Julius Dreisig", "Zeus X Crona"],
    "Manga Saint Hilare x Lewi B": ["Manga Saint Hilare", "Lewi B"],
    "NITTI & Kompany": ["NITTI", "Kompany"],
    "PHARAOH X Boulevard Depo": ["PHARAOH", "Boulevard Depo"],
    "Rasco & DJ Jonay": ["Rasco", "DJ Jonay"],
    "sapientdream & Slushii": ["sapientdream", "Slushii"],
    "Skism & Megalodon & Habstrakt": ["Skism", "Megalodon", "Habstrakt"],
    "Timmies & Nineteen95": ["Timmies", "Nineteen95"],
    "Tom Tykwer, Johnny Klimek & Reinhold Heil": ["Tom Tykwer", "Johnny Klimek", "Reinhold Heil"],
    "vowl., Sace & Silent Boy": ["vowl.", "Sace", "Silent Boy"],
}

# Typos and folder-name noise.
rename = {
    "Cheslea Grin": "Chelsea Grin",
    "Fro Rida": "Flo Rida",
    "Yeah Yeah Yeah's": "Yeah Yeah Yeahs",
    "Gigi D`Agostino": "Gigi D'Agostino",
    "Hi I`m Ghost": "Hi I'm Ghost",
    "P.O.D. (Payable On Death)": "P.O.D.",
    "Dope DOD.O": "Dope D.O.D.",
    "Ill Nino": "Ill Niño",
    "In VIce Versa": "In Vice Versa",
    "Mefjus [FLAC]": "Mefjus",
    "Subways": "The Subways",
}

names = sorted((d for d in os.listdir(src) if not d.startswith(".") and os.path.isdir(os.path.join(src, d))), key=str.casefold)
seen, result = set(), []
for d in names:
    if d in skip:
        continue
    for n in split.get(d, [rename.get(d, d)]):
        if n.casefold() not in seen:
            seen.add(n.casefold())
            result.append(n)
stale = [k for k in [*skip, *split, *rename] if k not in names]
if stale:
    print("warning: no such folders:", ", ".join(stale), file=sys.stderr)

with open(out, "w", encoding="utf-8") as f:
    f.write(f"# Artists from {src}: one per line, generated for Music Tracker import\n")
    f.write("\n".join(result) + "\n")
print(len(names), "folders ->", len(result), "artists")
