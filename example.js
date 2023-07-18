function need(a, ipPre, is4) {
    if (!a?.country?.iso_code == "CN") {
        return {
            "tag": "",
            "need": false
        }
    }
    if (is4) {
        return {
            "tag": "cn",
            "need": true
        }
    }
    return {
        "tag": "cn6",
        "need": true
    }
}