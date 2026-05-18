#include "testlib.h"
#include <vector>
#include <algorithm>
using namespace std;

int main(int argc, char* argv[]) {
    registerTestlibCmd(argc, argv);
    setName("unordered_checker");
    
    vector<string> ja, pa;
    while (!ans.seekEof()) ja.push_back(ans.readToken());
    while (!ouf.seekEof()) pa.push_back(ouf.readToken());
    
    if (ja.size() != pa.size())
        quitf(_wa, "expected %zu token(s), found %zu", ja.size(), pa.size());
    
    sort(ja.begin(), ja.end());
    sort(pa.begin(), pa.end());
    
    for (size_t i = 0; i < ja.size(); i++)
        if (ja[i] != pa[i])
            quitf(_wa, "token %zu differs after sorting: expected '%s', found '%s'", 
                  i, ja[i].c_str(), pa[i].c_str());
    
    quitf(_ok, "%zu token(s), order ignored", ja.size());
}
