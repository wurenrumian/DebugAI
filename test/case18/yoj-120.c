#include <stdio.h>
#define N 105

int ans[N], a[N], n;
int step;

void my_sort(int l, int r) {
    if (l == r) return;
    int pos = l;
    for (int i = l + 1; i <= r; ++i)
        if (a[i] < a[pos]) pos = i;
    if (pos != l) {
        step += 1;
        int tmp;
        tmp = a[l]; a[l] = a[pos]; a[pos] = tmp;
        my_sort(l + 1, r);
        printf ("%d<->%d:", l, pos);
        for (int i = 1; i <= n; ++i)
            printf ("%d ", a[i]);
        printf ("\n");
        step -= 1;
    }
}

int main () {
    scanf ("%d", &n);
    for (int i = 1; i <= n; ++i) {
        scanf ("%d", &a[i]);
    }
    my_sort(1, n);
    printf ("Total steps:%d\n", step);
    printf ("Right order:");
    for (int i = 1; i <= n; ++i)
        printf ("%d ", ans[i]);
    printf ("\n");
    return 0;
}