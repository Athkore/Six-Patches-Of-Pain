windows:	
	go generate 
	go build -o build\\Six-Patches-of-Pain.exe six_patches_of_pain.go 

linux:
	go build -o ./build/Six-Patches-of-Pain six_patches_of_pain.go 

mac:
	go build -o ./build/Six-Patches-of-Pain six_patches_of_pain.go 

get:
	go get github.com/cheggaaa/pb/v3
	go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
	go get github.com/Athkore/go-xdelta@b2df94f9b7a7fbf2493e4f47ae668bf295a6d264

clean:
	rm -rf ./build
