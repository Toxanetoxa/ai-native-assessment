#!/bin/sh
set -eu

# Only the compiled site is tracked in the public Pages repository.
checkout=$(mktemp -d)
trap 'rm -rf "$checkout"' EXIT
git clone --depth 1 git@github.com:Toxanetoxa/ai-native-assessment-demo.git "$checkout"

rm -rf "$checkout/assets"
cp -R web/dist/assets "$checkout/assets"
cp web/dist/index.html "$checkout/index.html"
touch "$checkout/.nojekyll"

git -C "$checkout" add -- .nojekyll index.html assets
if git -C "$checkout" diff --cached --quiet; then
  echo "На GitHub уже опубликована эта сборка."
  exit 0
fi
git -C "$checkout" commit -m "Обновить демо возвратов"
git -C "$checkout" push origin HEAD:main
