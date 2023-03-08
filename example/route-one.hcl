Kind = "http-route"
Name = "api-gateway-route"
Rules = [
  {
    Services = [
      {
        Name = "http-1"
      },
      {
        Name = "http-2"
      },
      {
        Name = "http-3"
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