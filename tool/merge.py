from pathlib import Path

SRC_DIR = Path("..")
OUT_FILE = Path("merged.txt")

def merge_files(src_dir: Path, out_file: Path):
    with out_file.open("w", encoding="utf-8") as out:
        for path in sorted(src_dir.rglob("*.go")):
            if not path.is_file():
                continue

            out.write(f"\n\n===== FILE: {path} =====\n\n")
            out.write(path.read_text(encoding="utf-8", errors="ignore"))

if __name__ == "__main__":
    merge_files(SRC_DIR, OUT_FILE)

