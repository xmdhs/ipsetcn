function need(a, ipPre, is4) {
    let n = false
    if (a?.country?.iso_code == "CN") {
        n = true
    }
    if (a?.country?.iso_code == "JP" && is4) {
        const l = ipPre.split(".")
        const l2 = Number(l[1])
        if (l[0] == "8" && l2 >= 208 && l2 <= 222) {
            n = true
        }
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