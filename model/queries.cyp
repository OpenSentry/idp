// Useful queries

// Find all consents given by an Identity
MATCH (i:Identity {sub:$sub})
MATCH (i)<-[:GrantedBy]-(r:Rule)-[:Grant]->(p:Policy)-[:Grant]->(a:Permission)
MATCH (n:Identity)-[:IsGranted]->(r)
return i, r, n, p, a

// Find all scopes exposed by an identity
MATCH (i:Identity {sub:$sub})-[:Exposes]->(p:Policy)-[:Grant]->(a:Permission)
return i, p, a
