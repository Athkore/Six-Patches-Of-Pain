pyinstaller --onefile --icon=rinnegan.ico six_patches_of_pain.py
Move-Item -Path dist/six_patches_of_pain.exe -Destination six_patches_of_pain.exe
tar.exe -a -c -f dist/windows.zip six_patches_of_pain.exe data/xdelta3.exe
Move-Item -Path six_patches_of_pain.exe -Destination dist/six_patches_of_pain.exe
tar.exe -a -c -f dist/mac.zip six_patches_of_pain.py
tar.exe -a -c -f dist/linux.zip six_patches_of_pain.py
