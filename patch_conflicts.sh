sed -i '/<<<<<<< HEAD/,/=======\|>>>>>>> origin\/main/d' collector/commands.go
sed -i 's/>>>>>>> origin\/main//g' collector/commands.go
