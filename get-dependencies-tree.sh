#!/bin/sh

ignore_pkgs="pipdeptree,pip-licenses,PTable,graphviz,pip,wheel,setuptools"

cat>get-deps.sh <<EOF
#!/bin/sh

apk update && apk add graphviz ttf-freefont

go get github.com/google/go-licenses
go get github.com/kisielk/godepgraph

cd code/peripheral-manager-usb
cp ../../LICENSE .

godepgraph -s github.com/nuvlabox/peripheral-manager-usb | dot -Tpng -o dependencies-tree.png
go-licenses csv . --stderrthreshold 3 > dependencies-licenses.txt
EOF

chmod +x get-deps.sh

docker run --entrypoint /bin/sh -v $(pwd):/deptree --workdir /deptree golang:alpine /deptree/get-deps.sh

rm get-deps.sh
