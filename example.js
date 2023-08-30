function need(a, ip, asn, is4) {
    const n = (() => {
        if (asn == 140633) {
            return false
        }
        if (a?.country?.iso_code == "CN") {
            return true
        }
        if (a?.country?.iso_code == "JP" && asn == 45102) {
            return true
        }
    })()

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
