#include "testlib.h"
#include <fstream>
#include <sstream>
using namespace std;

string readFile(const char* path) {
    ifstream f(path, ios::binary);
    if (!f) quitf(_fail, "cannot open file: %s", path);
    stringstream buf;
    buf << f.rdbuf();
    return buf.str();
}

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
    setName("strict_checker");

    string ja = readFile(argv[2]); // answer (expected)
    string pa = readFile(argv[3]); // output (participant)

    // Exact byte-for-byte match
    if (ja == pa) {
        quitf(_ok, "exact match");
    }

    // If bytes differ, check if tokens are the same (PE detection)
    // Reset streams to compare tokens
    ans.reset();
    ouf.reset();

    int n = 0;
    while (!ans.seekEof() && !ouf.seekEof()) {
        string t1 = ans.readToken();
        string t2 = ouf.readToken();
        if (t1 != t2)
            quitf(_wa, "token %d differs: expected '%s', found '%s'", ++n, t1.c_str(), t2.c_str());
        ++n;
    }

    int extra = 0;
    while (!ouf.seekEof()) { ouf.readToken(); extra++; }
    if (extra > 0) quitf(_wa, "participant output contains %d extra token(s)", extra);

    int missing = 0;
    while (!ans.seekEof()) { ans.readToken(); missing++; }
    if (missing > 0) quitf(_wa, "participant output misses %d token(s)", missing);

    // Tokens match exactly but raw files differ → Presentation Error
    quitf(_pe, "tokens match but presentation differs (whitespace/line endings)");
}
