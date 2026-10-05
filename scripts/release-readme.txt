VRChat Asset Manager
====================

A personal catalog of your VRChat assets (avatars, hair, outfits, gimmicks...):
what you bought, what it looks like, where it is on your PC, and its BOOTH page.


START
-----
1. Unzip this folder anywhere you like (for example Documents\VRChatAssetManager).
   Do not run it from inside the zip.
2. Double-click VRChatAssetManager.exe.
3. Your browser opens the app. If it does not, open http://127.0.0.1:47380

The first time, Windows may show "Windows protected your PC".
The app is not code-signed; click "More info" -> "Run anyway".


STOP
----
Close the black console window. The app only runs while that window is open.

Double-clicking the exe again while it is running just opens the browser.


FIRST STEPS
-----------
- Scan & Review (left sidebar) -> add your asset folder (e.g. D:\VRChat Assets)
  -> Scan. Every found asset becomes a draft you can accept, edit or ignore.
- Or add assets one by one with "Add Asset".
- "Fetch from BOOTH" fills name, author, category and picture from a BOOTH link.

The app never moves, renames or deletes your asset files.


YOUR DATA
---------
Everything the app stores is in the "data" folder next to the exe:

    data\app.db       your library
    data\previews\    preview pictures
    data\backups\     automatic backups made before updates

To back up: copy the "data" folder.
To move to another PC: copy the whole VRChatAssetManager folder.


UPDATE
------
Close the app, replace VRChatAssetManager.exe with the new one, keep "data".
The database is backed up automatically before it is upgraded.


PROBLEMS
--------
- "Cannot listen on ...": another program uses port 47380. Close it, or start
  the app from a command prompt with:  set PORT=47381 && VRChatAssetManager.exe
- Nothing happens / the window closes at once: run it from a command prompt to
  read the error message.
