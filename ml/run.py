"""
Flask entry point for the ML sidecar.

Run with any of:
    python -m ml.run           # from repo root — cleanest, recommended
    python ml/run.py           # from repo root
    python run.py              # from inside ml/
"""
import sys
from pathlib import Path

# `ml/app` imports as `from ml.app import ...` (an absolute import of the
# top-level `ml` package), so the repo root — ml/'s parent — must be on
# sys.path. Running "python ml/run.py" or "python run.py" from inside ml/
# only puts ml/run.py's own directory on sys.path, not the repo root, so
# the import fails without this. python -m ml.run doesn't need this (it
# already puts the repo root on sys.path) but the insert is harmless there.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from ml.app import create_app  # noqa: E402
from ml.app.config import Config  # noqa: E402

app = create_app()

if __name__ == "__main__":
    app.run(host=Config.HOST, port=Config.PORT, debug=Config.DEBUG)