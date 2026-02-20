#include <iostream>
#include <cstring>
#include <stdio.h>
using namespace std;
void f(int h[], int hh)
{
    for (int ii = 1; ii < hh; ii++)
    {
        h[ii - 1] = -1 * h[ii];
    }
}
void ff(int a1[], const int a2[], const int a3[], int bi, int sm)
{
    int ii, iii;
    for (ii = sm, iii = bi; ii >= 1; ii--, iii--)
    {
        a1[iii - 1] = a2[iii - 1] + a3[ii - 1];
    }
    for (iii; iii >= 1; iii--)
    {
        a1[iii - 1] = a2[iii - 1];
    }
}
int main()
{
    char p;
    scanf("%s\n", &p);
    char a[2001];
    char b[2001];
    cin.getline(a, 2001, '\n');
    cin.getline(b, 2001, '\n');
    int m = strlen(a), n = strlen(b);
    int aa[2001];
    int bb[2001];
    int i;
    for (i = 0; i < m; i++)
    {
        aa[i] = a[i] - 48;
    }
    for (i = 0; i < n; i++)
    {
        bb[i] = b[i] - 48;
    }
    if (aa[0] < 0)
    {
        f(aa, m);
        m--;
    }
    if (bb[0] < 0)
    {
        f(bb, n);
        n--;
    }
    if (p == '-')
    {
        for (i = 0; i < n; i++)
        {
            bb[i] *= -1;
        }
    }
    int cc[2001];
    int t = 0;
    if (m >= n)
    {
        ff(cc, aa, bb, m, n);
        while (cc[t] == 0)
        {
            t++;
        }
        if (cc[t] > 0)
        {
            for (i = m - 1; i >= t + 1; i--)
            {
                if (cc[i] < 0)
                {
                    cc[i - 1] -= 1;
                    cc[i] += 10;
                }
                if (cc[i] >= 10)
                {
                    cc[i - 1] += 1;
                    cc[i] -= 10;
                }
            }
            while (cc[t] == 0)
            {
                t++;
            }
            for (i = t; i <= m - 1; i++)
            {
                cout << cc[i];
            }
        }
        if (cc[t] < 0)
        {
            for (i = t; i <= m - 1; i++)
            {
                cc[i] *= -1;
            }
            for (i = m - 1; i >= t + 1; i--)
            {
                if (cc[i] < 0)
                {
                    cc[i - 1] -= 1;
                    cc[i] += 10;
                }
                if (cc[i] >= 10)
                {
                    cc[i - 1] += 1;
                    cc[i] -= 10;
                }
            }
            cout << '-';
            while (cc[t] == 0)
            {
                t++;
            }
            for (i = t; i <= m - 1; i++)
            {
                cout << cc[i];
            }
        }
    }
    if (n > m)
    {
        ff(cc, bb, aa, n, m);
        while (cc[t] == 0)
        {
            t++;
        }
        if (cc[t] > 0)
        {
            for (i = n - 1; i >= t + 1; i--)
            {
                if (cc[i] < 0)
                {
                    cc[i - 1] -= 1;
                    cc[i] += 10;
                }
                if (cc[i] >= 10)
                {
                    cc[i - 1] += 1;
                    cc[i] -= 10;
                }
            }
            while (cc[t] == 0)
            {
                t++;
            }
            for (i = t; i <= n - 1; i++)
            {
                cout << cc[i];
            }
        }
        if (cc[t] < 0)
        {
            for (i = t; i <= n - 1; i++)
            {
                cc[i] *= -1;
            }
            for (i = n - 1; i >= t + 1; i--)
            {
                if (cc[i] < 0)
                {
                    cc[i - 1] -= 1;
                    cc[i] += 10;
                }
                if (cc[i] >= 10)
                {
                    cc[i - 1] += 1;
                    cc[i] -= 10;
                }
            }
            cout << '-';
            while (cc[t] == 0)
            {
                t++;
            }
            for (i = t; i <= n - 1; i++)
            {
                cout << cc[i];
            }
        }
    }
    return 0;
}