from pathlib import Path

SRC_DIR = Path("..")
OUT_FILE = Path("merged.txt")

INCLUDE_GLOBS = [
    "*.go",
    "*.mk",
    "*.proto"
]

INCLUDE_FILENAMES = {
    "Makefile",
    "go.mod",
    "go.sum",
}

def should_include(path: Path) -> bool:
    name = path.name
    if name in INCLUDE_FILENAMES:
        return True
    for pat in INCLUDE_GLOBS:
        if path.match(pat):
            return True
    return False

def merge_files(src_dir: Path, out_file: Path):
    files = []
    for p in src_dir.rglob("*"):
        if not p.is_file():
            continue
        if should_include(p):
            files.append(p)

    files = sorted(set(files), key=lambda x: str(x))

    with out_file.open("w", encoding="utf-8") as out:
        for path in files:
            out.write(f"\n\n===== FILE: {path} =====\n\n")
            out.write(path.read_text(encoding="utf-8", errors="ignore"))

if __name__ == "__main__":
    merge_files(SRC_DIR, OUT_FILE)
