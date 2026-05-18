#include "testlib.h"
using namespace std;

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
    setName("default_checker");
    
    int n = 0;
    while (!ans.seekEof() && !ouf.seekEof()) {
        string ja = ans.readToken();
        string pa = ouf.readToken();
        if (ja != pa)
            quitf(_wa, "token %d differs: expected '%s', found '%s'", ++n, ja.c_str(), pa.c_str());
        ++n;
    }
    
    int extra = 0;
    while (!ouf.seekEof()) { ouf.readToken(); extra++; }
    if (extra > 0) quitf(_wa, "participant output contains %d extra token(s)", extra);
    
    int missing = 0;
    while (!ans.seekEof()) { ans.readToken(); missing++; }
    if (missing > 0) quitf(_wa, "participant output misses %d token(s)", missing);
    
    if (n == 0) quitf(_ok, "empty output is acceptable");
    quitf(_ok, "%d token(s)", n);
}
