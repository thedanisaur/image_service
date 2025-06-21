# Movie Sunday Image Service

### Curl Commands
For my Alzheimer's brain

```
curl -X GET http://localhost:7331/images/men_in_black_1

curl -X POST http://localhost:7331/images -H 'Content-Type: application/json' -d '{ "movie_title":"Men in Black", "movie_name":"men_in_black_1", "series_name":"men_in_black" }'
```

find . -iname "*.webp" -delete
find . -iname "*.webp" -print0 | xargs -0 du -ch | tail -n 1