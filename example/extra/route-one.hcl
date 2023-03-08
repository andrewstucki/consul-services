Kind = "http-route"
Name = "api-gateway-route"
Rules = [
  {
    Services = [
      {
        Name = "http-1"
      }
    ]
  }  
]

Parents = [
  {
    SectionName = "listener-one"
    Name        = "api"
  },
  {
    SectionName = "listener-two"
    Name        = "api"
  },
]