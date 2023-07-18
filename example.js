function need(a, ip, asn, is4) {
    let n = false
    if (a?.country?.iso_code == "CN") {
        n = true
    }
    if (a?.country?.iso_code == "JP" && asn == 45102) {
        n = true
    }

    if (!n) {
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
