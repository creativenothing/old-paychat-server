package main

import (
    "encoding/json"
)

var users map[int]User = map[int]User{}

type User struct{
    // Name
    username string

    // Unique id
    userid int


}

func readUser(v map[interface{}]interface{}) User {
    userJSON := v["user"].(string)
    userParsed := map[string]interface{}{}
    json.Unmarshal([]byte(userJSON), &userParsed)

    user := User{
        username: userParsed["username"].(string),
        userid: int(userParsed["userid"].(float64)),
    }

    return user
}

func writeUser(v map[interface{}]interface{},u User){
    message, _ := json.Marshal(map[string]interface{}{
        "userid": u.userid,
        "username": u.username,
    })
    v["user"] = string(message)
}
