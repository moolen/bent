const express = require('express')
const auth = require('basic-auth')
const app = express()
const port = 8080

app.use((req, res, next) => {
  console.log(req.path)
  console.log(req.headers)

  if (req.path == "/healthz"){
    res.status(200)
    res.end('OK')
    return
  }

  const creds = auth(req)
  if (!creds) {
    res.status(401)
    res.end('Unauthorized')
  }

  if(creds.name === process.env.AUTH_USER && creds.pass === process.env.AUTH_PASS) {
    res.status(200)
    res.end()
  } else {
    res.status(401)
    res.end('Unauthorized')
  }
})

app.listen(port, () => console.log(`authz listening on port ${port}!`))
