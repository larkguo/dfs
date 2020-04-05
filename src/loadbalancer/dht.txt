/*
  xlator_t *dht_layout_search (xlator_t *this, dht_layout_t *layout, const char *name) 
  -> int dht_hash_compute (xlator_t *this, int type, const char *name, uint32_t *hash_p) 
  -> int dht_hash_compute_internal (int type, const char *name, uint32_t *hash_p)  
  -> uint32_t  gf_dm_hashfn (const char *msg, int len)  
  -> static int dm_round (int rounds, uint32_t *array, uint32_t *h0, uint32_t *h1)
*/
#include <stdint.h>
#include <stdlib.h>
#include <stdbool.h>
#include <regex.h>

uint32_t SuperFastHash (const char * data, int32_t len);
uint32_t gf_dm_hashfn (const char *msg, int len);
uint32_t ReallySimpleHash (char *path, int len);

#define get16bits(d) (*((const uint16_t *) (d)))
#define DM_DELTA 0x9E3779B9
#define DM_FULLROUNDS 10        /* 32 is overkill, 16 is strong crypto */
#define DM_PARTROUNDS 6         /* 6 gets complete mixing */

uint32_t
ReallySimpleHash (char *path, int len)
{
        uint32_t        hash = 0;
        for (;len > 0; len--)
                hash ^= (char)path[len];

        return hash;
}

/*
  This is apparently the "fastest hash function for strings".
  Written by Paul Hsieh <http://www.azillionmonkeys.com/qed/hash.html>
*/
/* In any case make sure, you return 1 */
uint32_t SuperFastHash (const char * data, int32_t len) {
        uint32_t hash = len, tmp;
        int32_t rem;

        if (len <= 1 || data == NULL) return 1;

        rem = len & 3;
        len >>= 2;

        /* Main loop */
        for (;len > 0; len--) {
                hash  += get16bits (data);
                tmp    = (get16bits (data+2) << 11) ^ hash;
                hash   = (hash << 16) ^ tmp;
                data  += 2*sizeof (uint16_t);
                hash  += hash >> 11;
        }

        /* Handle end cases */
        switch (rem) {
        case 3: hash += get16bits (data);
                hash ^= hash << 16;
                hash ^= data[sizeof (uint16_t)] << 18;
                hash += hash >> 11;
                break;
        case 2: hash += get16bits (data);
                hash ^= hash << 11;
                hash += hash >> 17;
                break;
        case 1: hash += *data;
                hash ^= hash << 10;
                hash += hash >> 1;
        }

        /* Force "avalanching" of final 127 bits */
        hash ^= hash << 3;
        hash += hash >> 5;
        hash ^= hash << 4;
        hash += hash >> 17;
        hash ^= hash << 25;
        hash += hash >> 6;

        return hash;
}

/* Davies-Meyer hashing function implementation
 */
static int
dm_round (int rounds, uint32_t *array, uint32_t *h0, uint32_t *h1)
{
        uint32_t sum = 0;
        int      n = 0;
        uint32_t b0  = 0;
        uint32_t b1  = 0;

        b0 = *h0;
        b1 = *h1;

        n = rounds;

        do {
                sum += DM_DELTA;
                b0  += ((b1 << 4) + array[0])
                        ^ (b1 + sum)
                        ^ ((b1 >> 5) + array[1]);
                b1  += ((b0 << 4) + array[2])
                        ^ (b0 + sum)
                        ^ ((b0 >> 5) + array[3]);
        } while (--n);

        *h0 += b0;
        *h1 += b1;

        return 0;
}

uint32_t
__pad (int len)
{
        uint32_t pad = 0;

        pad = (uint32_t) len | ((uint32_t) len << 8);
        pad |= pad << 16;

        return pad;
}

uint32_t
gf_dm_hashfn (const char *msg, int len)
{
        uint32_t  h0 = 0x9464a485;
        uint32_t  h1 = 0x542e1a94;
        uint32_t  array[4];
        uint32_t  pad = 0;
        int       i = 0;
        int       j = 0;
        int       full_quads = 0;
        int       full_words = 0;
        int       full_bytes = 0;
        uint32_t *intmsg = NULL;
        int       word = 0;


        intmsg = (uint32_t *) msg;
        pad = __pad (len);

        full_bytes   = len;
        full_words   = len / 4;
        full_quads   = len / 16;

        for (i = 0; i < full_quads; i++) {
                for (j = 0; j < 4; j++) {
                        word     = *intmsg;
                        array[j] = word;
                        intmsg++;
                        full_words--;
                        full_bytes -= 4;
                }
                dm_round (DM_PARTROUNDS, &array[0], &h0, &h1);
        }

        for (j = 0; j < 4; j++) {
                if (full_words) {
                        word     = *intmsg;
                        array[j] = word;
                        intmsg++;
                        full_words--;
                        full_bytes -= 4;
                } else {
                        array[j] = pad;
                        while (full_bytes) {
                                array[j] <<= 8;
                                array[j] |= msg[len - full_bytes];
                                full_bytes--;
                        }
                }
        }
        dm_round (DM_FULLROUNDS, &array[0], &h0, &h1);

        return h0 ^ h1;
}

static bool 
dht_munge_name (const char *original, char *modified,size_t len, regex_t *re)
{
        regmatch_t  matches[2] = {{0}, };
        size_t      new_len    = 0;
        int         ret        = 0;

        ret = regexec(re, original, 2, matches, 0);
        if (ret != REG_NOMATCH) {
                if (matches[1].rm_so != -1) {
                        new_len = matches[1].rm_eo - matches[1].rm_so;
                        /* Equal would fail due to the NUL at the end. */
                        if (new_len < len) {
                                memcpy(modified,original+matches[1].rm_so,new_len);
                                modified[new_len] = '\0';
                                return true;
                        }
                }
        }

        /* This is guaranteed safe because of how the dest was allocated. */
        strcpy(modified, original);
        return false;
}

/*
gcc dht.c -o dht

./dht  .fio.sh
using regex '^\.(.+)\.[^.]+$'
dht_munge_name '.fio.sh' -> 'fio'
gf_dm_hashfn('fio')	= 2164401422	(0x8102250e)
*/
void 
main(int argc, char ** argv)
{
    int i = 0;
    char  *name = NULL;
    char  *rsync_friendly_name = NULL;
    size_t len = 0;
    bool munged = false;

    for (i=1; i < argc; i++){
        uint32_t hash = 0;
        regex_t re;
        char *temp_str = NULL;

        // init regex
        temp_str = "^\\.(.+)\\.[^.]+$";
        if (regcomp(&re, temp_str, REG_EXTENDED) == 0) {
                printf("using regex '%s'\n", temp_str);
        }

        //change name
        len = strlen(argv[i])+1;
        name = alloca(len);
        strncpy(name, argv[i],len);
        rsync_friendly_name = alloca(len);
        munged = dht_munge_name(name, rsync_friendly_name, len, &re);
        if (!munged) {
                rsync_friendly_name = (char *)name;
        }
        printf("dht_munge_name '%s' -> '%s'\n",name,rsync_friendly_name);

        // dht hash
        hash = gf_dm_hashfn(rsync_friendly_name, strlen(rsync_friendly_name));
        printf("gf_dm_hashfn('%s')\t= %u\t(0x%x)\n",rsync_friendly_name,hash,hash);
    }
}
