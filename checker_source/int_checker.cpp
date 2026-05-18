#include "testlib.h"
using namespace std;

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
    setName("int_checker");
    
    int n = 0;
    while (!ans.seekEof() && !ouf.seekEof()) {
        long long ja = ans.readLong();
        long long pa = ouf.readLong();
        if (ja != pa)
            quitf(_wa, "integer %d differs: expected %lld, found %lld", ++n, ja, pa);
        ++n;
    }
    
    int extra = 0;
    while (!ouf.seekEof()) { ouf.readLong(); extra++; }
    if (extra > 0) quitf(_wa, "participant output contains %d extra integer(s)", extra);
    
    int missing = 0;
    while (!ans.seekEof()) { ans.readLong(); missing++; }
    if (missing > 0) quitf(_wa, "participant output misses %d integer(s)", missing);
    
    quitf(_ok, "%d integer(s)", n);
}
