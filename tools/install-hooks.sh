#!/bin/bash
# tools/install-hooks.sh

HOOKS_DIR=".git/hooks"
POST_COMMIT_SRC="tools/hooks/post-commit"
POST_COMMIT_DEST="$HOOKS_DIR/post-commit"

echo "🔧 Instalando Git Hooks para simulación CI/CD local..."

# 1. Ensure .git/hooks exists
mkdir -p "$HOOKS_DIR"

# 2. Check if post-commit already exists
if [ -f "$POST_COMMIT_DEST" ]; then
    echo "⚠️  Ya existe un hook 'post-commit'. Se hará un backup."
    mv "$POST_COMMIT_DEST" "$POST_COMMIT_DEST.bak"
fi

# 3. Create symlink or copy
# Using copy to avoid permissions issues if the repo is moved
cp "$POST_COMMIT_SRC" "$POST_COMMIT_DEST"
chmod +x "$POST_COMMIT_DEST"

echo "✅ Hook 'post-commit' instalado correctamente."
echo "👉 Ahora cada vez que hagas 'git commit', se desplegará automáticamente en tu entorno local (Lambda/EKS)."
