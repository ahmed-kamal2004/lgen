go run main.go http --destination "http://localhost:8000/SendMessage" --conc 10 --reqn 100 --method POST --reqb_path test-scripts/body.json

go run main.go http --destination "http://localhost:8000/SendMessage" --conc 10 --reqn 100 --method POST --reqb_path test-scripts/body.json --reqn 10000
