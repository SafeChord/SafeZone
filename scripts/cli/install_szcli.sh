# ---------------- Step 1: Set Variables ----------------------
SCRIPT_PATH=$(pwd)/scripts/cli/local_run.sh
INSTANCE_NAME="safezone-cli-daemon"

# check script is exist or not
if [ ! -f "$SCRIPT_PATH" ]; then
  echo "[ERROR] local_run.sh not found at $SCRIPT_PATH. Please run this script from the SafeZone directory."
  exit 1
fi

# ----------- PHASE 0: stop the previous version daemon container and remove --------------
# This phase ensures that any previous instances of the CLI daemon are stopped and removed..
# Check if the CLI daemon is running, and stop it if it is.
if docker ps -q --filter "name=$INSTANCE_NAME" | grep -q .; then
  echo "[PHASE 0] Stopping running $INSTANCE_NAME..."
  docker stop $INSTANCE_NAME
fi
# Check if the CLI daemon image exists, and remove it if it does.
if docker ps -a --filter "name=$INSTANCE_NAME" --filter "status=exited" | grep -q $INSTANCE_NAME; then
  echo "[PHASE 0] Found exited $INSTANCE_NAME, removing..."
  docker rm $INSTANCE_NAME
fi

# --------------- PHASE 1: CLI Global Installation ------------
# The following section sets up the 'szcli' command as a global shell command.
# This avoids the need to set up an alias for every shell session.


# ~/.bin is used instead of system-wide directories (like /usr/local/.bin) to avoid requiring sudo/root privileges.
# This is a common cross-platform (Linux/macOS) convention for user-level CLI tools.
mkdir -p ~/.bin

# Symlink local_run.sh to ~/.bin/szcli.
# This allows the user to run 'szcli' from any location and in any shell session.
ln -sf "$SCRIPT_PATH" ~/.bin/szcli
chmod +x ~/.bin/szcli

# If ~/.bin is not in the user's PATH, append it to ~/.bashrc to ensure 'szcli' is always available.
# This is necessary because some Linux distributions do not add ~/.bin to PATH by default.
if ! echo "$PATH" | grep -q "$HOME/.bin" ; then
  echo 'export PATH="$HOME/.bin:$PATH"' >> ~/.bashrc
  echo '[INFO] Added ~/.bin to your PATH in ~/.bashrc. The change will take effect after you restart your shell or run: source ~/.bashrc'
fi

echo "[SUCCESS] 'szcli' is now available as a global CLI command!"
echo "Please open a new shell session, or manually run: export PATH=\"\$HOME/.bin:\$PATH\""
echo "To remove the command, simply run: rm ~/.bin/szcli"