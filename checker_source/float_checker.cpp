#include "testlib.h"
#include <cmath>
#include <cstdlib>
using namespace std;

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
    setName("float_checker");
    
    double EPS = 1e-9;
    if (argc > 4) EPS = atof(argv[4]);
    
    int n = 0;
    while (!ans.seekEof() && !ouf.seekEof()) {
        double ja = ans.readDouble();
        double pa = ouf.readDouble();
        
        if (!doubleCompare(ja, pa, EPS))
            quitf(_wa, "number %d differs: expected %.10f, found %.10f (eps=%.2e)", 
                  ++n, ja, pa, EPS);
        ++n;
    }
    
    int extra = 0;
    while (!ouf.seekEof()) { ouf.readDouble(); extra++; }
    if (extra > 0) quitf(_wa, "participant output contains %d extra number(s)", extra);
    
    int missing = 0;
    while (!ans.seekEof()) { ans.readDouble(); missing++; }
    if (missing > 0) quitf(_wa, "participant output misses %d number(s)", missing);
    
    quitf(_ok, "%d number(s)", n);
}
