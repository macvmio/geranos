package zstd

import (
	"bytes"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand"
	"testing"
)

// Test_ReadCloser verifies that ReadCloser compresses the input correctly.
func Test_ReadCloser(t *testing.T) {
	// Create a buffer filled with 327680 bytes of zeroes as input
	inputData := bytes.Repeat([]byte{0}, 327680)
	input := io.NopCloser(bytes.NewReader(inputData))

	// Use ReadCloser to compress the data
	compressedReader := ReadCloser(input)
	defer compressedReader.Close()

	h, n, err := v1.SHA256(compressedReader)
	require.NoError(t, err)
	assert.Equal(t, v1.Hash{
		Algorithm: "sha256",
		Hex:       "ffc1459590bc9dee536a1fb0a6fee5b31beefa35348488541c343c1d3af41f5d",
	}, h)
	assert.Equal(t, int64(78), n)
}

type TestInputData struct {
	seed int64
	name string
	data []byte
}

// generateTestInputs creates test inputs of specified sizes with ranges of zeroes using a seeded RNG.
func generateTestInput(seed int64, size int) TestInputData {
	rng := rand.New(rand.NewSource(seed))
	data := make([]byte, size)
	_, err := rng.Read(data)
	if err != nil {
		panic("failed to generate random data")
	}

	// Introduce ranges of zeroes
	zeroRangeStart := size / 4
	zeroRangeEnd := zeroRangeStart + size/8
	for i := zeroRangeStart; i < zeroRangeEnd && i < size; i++ {
		data[i] = 0
	}

	return TestInputData{
		seed: seed,
		name: fmt.Sprintf("input_%dkb", size/1024),
		data: data,
	}
}

// computeExpectedHashes computes and prints the SHA256 hashes of the compressed outputs.
func computeExpectedHashes(t *testing.T, inputs []TestInputData) {
	for _, input := range inputs {
		r := io.NopCloser(bytes.NewReader(input.data))
		compressedReader := ReadCloser(r)
		defer compressedReader.Close()

		hash, _, err := v1.SHA256(compressedReader)
		require.NoError(t, err)
		fmt.Printf("{ name: \"%s\", seed: %d, dataSize: %d, expectedHash: \"%s\" },\n", input.name, input.seed, len(input.data), hash.Hex)
	}
}

// Test_GenerateExpectedHashes generates the expected hashes for the test inputs.
func Test_GenerateExpectedHashes(t *testing.T) {
	t.Skip("this is generator. Run it only if compression changes are intentional")
	// Initialize random number generators with fixed seeds
	seed := int64(42)
	rngSizes := rand.New(rand.NewSource(seed))

	// Generate 100 random sizes between 128 KB and 1024 KB
	inputs := make([]TestInputData, 100)
	for i := 0; i < 100; i++ {
		size := rngSizes.Intn(1024*1024-128*1024+1) + 128*1024
		seed2 := rngSizes.Int63()
		inputs[i] = generateTestInput(seed2, size)
	}

	computeExpectedHashes(t, inputs)
}

func Test_CompressionStability(t *testing.T) {
	testCases := []struct {
		name         string
		dataSize     int
		seed         int64
		expectedHash string
	}{
		{name: "input_216kb", seed: 608747136543856411, dataSize: 221512, expectedHash: "481aa9c3b9517a379e6c6eea44e3807b54f4497f96164c41104b14ad146dc88d"},
		{name: "input_955kb", seed: 1926012586526624009, dataSize: 978175, expectedHash: "501dfe266056749de52007ac04d57dd51ee5606770b4ebc992bc5b6297758ba8"},
		{name: "input_629kb", seed: 3534334367214237261, dataSize: 644985, expectedHash: "7bddaec6b0d02040e604fe2b2262be2bbb934781f6ac2e8eb510eca0f96e8de0"},
		{name: "input_661kb", seed: 3545887102062614208, dataSize: 676919, expectedHash: "48a72e4f95477e7af842a34d18907d79d9148da7416e35627039226b2405109b"},
		{name: "input_613kb", seed: 5961769557461764184, dataSize: 628720, expectedHash: "8a3b54c901900857ce64d80a279ea9ea31942a94373e50bcf27d96ad2329081f"},
		{name: "input_836kb", seed: 2010156224608899041, dataSize: 856896, expectedHash: "18a619051bf5f8e24b4596895e13b50427ea62fae3c9f3ccb41a0bb932dbd8b6"},
		{name: "input_614kb", seed: 1118224062840270781, dataSize: 629657, expectedHash: "b546041963ef8966da28ed643c6bc35a34d915b62d1b25055161dc02ae3c983c"},
		{name: "input_274kb", seed: 4299969443970870044, dataSize: 280954, expectedHash: "2d60f762960ea8f380a2fd98b6eeff8c5c8027ccc22c51475cb3e31a74032e7d"},
		{name: "input_433kb", seed: 6603068123710785442, dataSize: 444289, expectedHash: "e3ddb62fdbf20bbaf9878dc2ead7a5ab67e0a9d9cf5fbac87f27d6333cb51e86"},
		{name: "input_169kb", seed: 8953870811860009499, dataSize: 173587, expectedHash: "b28748b935ca1a3e64975499399918518e0aedc2ef4dab6e82397e051d81dc8b"},
		{name: "input_400kb", seed: 3137486211269163098, dataSize: 410259, expectedHash: "07a33f34fe177e03402b40bb6d92ae1d44f84246011a3cf5b671f28e36f40096"},
		{name: "input_828kb", seed: 2094832826273809275, dataSize: 848166, expectedHash: "564229d09932819bc77b330c77d86003b5500ea97bf97b8bc1f9a6d6ac78bac2"},
		{name: "input_456kb", seed: 387177823285674209, dataSize: 467481, expectedHash: "b85e524f2c11dfb1853021c3a3d9d5e4471ba7b8923519ccf013989eb83c44e2"},
		{name: "input_318kb", seed: 1198686371618757660, dataSize: 325729, expectedHash: "8acf081dd7f334020d76bd5cfdd718d2ca32850f020f425084779d390bf65e68"},
		{name: "input_156kb", seed: 1947031852228291041, dataSize: 159804, expectedHash: "52eff7e67dcc5e1a2647015f25ac1051862c36ce90c909ba06a0039b688b9bdb"},
		{name: "input_842kb", seed: 6656619927997724782, dataSize: 862259, expectedHash: "deee4506e852a98f1750726e9aa728a3dbe7ddd1f89e9bb7f3df3377fc18afe3"},
		{name: "input_335kb", seed: 9198671517108524046, dataSize: 343242, expectedHash: "a2d403f48831895e6fad1850c5ec5ec610f8f50cebefba6aef8b98b2c43ac5b9"},
		{name: "input_585kb", seed: 8978767854375171209, dataSize: 599881, expectedHash: "2f31124e3fd54dfb90cdbdfc9fff3b6efd683ef656c57d1f618959cf2861bc4f"},
		{name: "input_580kb", seed: 7894140303635748408, dataSize: 594086, expectedHash: "5f01c49c9548e5f0bf29121eddf0f8909a89177b5c9ec3735513de10430592b1"},
		{name: "input_830kb", seed: 5785305945910038487, dataSize: 850874, expectedHash: "4c6a510c41238ba00ceb1ec5077065da19fdc7259b1f9fc7cb09ec235ad9d506"},
		{name: "input_603kb", seed: 6972490225919430754, dataSize: 618112, expectedHash: "dd236d4feeec82c5801c73e845b356602289d763039c560c1412437355f0f504"},
		{name: "input_443kb", seed: 524317412587104192, dataSize: 454411, expectedHash: "fa106bd96a077a5b8e8c3d6d02ec3d4321b17f2438333fa655d9488a0454e4b9"},
		{name: "input_910kb", seed: 5274380952544653919, dataSize: 932368, expectedHash: "433760fed25eff2d56b4723099ba1b00e5b1bbde0bd8ca1df519ebd00a805efa"},
		{name: "input_542kb", seed: 5013667937579866784, dataSize: 556001, expectedHash: "3ed1b3582b670dfbb438d47931897c9ef7260d17729674fde397436f2e9af35c"},
		{name: "input_424kb", seed: 7276889728610971305, dataSize: 434486, expectedHash: "ca00549c08c51b13addda005246d2d2d93eb56dd1ae8b2b499fbb8558b1b25fe"},
		{name: "input_604kb", seed: 1628829379025336882, dataSize: 619365, expectedHash: "300db4126905f47ec0983f9306daf6210dda3bd1027105b3a6bbe519dffd25e4"},
		{name: "input_194kb", seed: 7317580557654915562, dataSize: 199201, expectedHash: "2d57f2a5437a9745a229fcc670291c4c35ecd83c9846afac13e97438074d0845"},
		{name: "input_220kb", seed: 2157255887366150544, dataSize: 225409, expectedHash: "45d6434c09be9791e0be8ce4e9964b0b34e649700c361662e783f2be37f98a91"},
		{name: "input_806kb", seed: 2628757933617101898, dataSize: 825710, expectedHash: "5cefaef64453ad4bdd35929f2a21e2f63dcf1c58a67e2d49aa3b0004bf7e9d01"},
		{name: "input_603kb", seed: 2388284803033957873, dataSize: 617903, expectedHash: "8618aa5f71cad9440e8f6ec0fcd21d280ab805f18e23dcee2184df92d2ad8e1b"},
		{name: "input_738kb", seed: 6317064433972108951, dataSize: 755886, expectedHash: "400f67c8c32389ca251a4fe5d7f2768b00a63835054374e8ac35e713094548e3"},
		{name: "input_332kb", seed: 6683509660637259755, dataSize: 340523, expectedHash: "f4082ceef1fa8c3ccd1af407fbc17a852d7592bc359c8c8bd207667eca0cb528"},
		{name: "input_972kb", seed: 2750659816549965649, dataSize: 995882, expectedHash: "d0f0981594609abb8c40d7832f9462f6a1400aedd15e4ee47bdd491f8b1b73da"},
		{name: "input_855kb", seed: 5328953735686644455, dataSize: 875998, expectedHash: "796851f7f21203078ed6d083d02ddef49bb93d9dc3d850675d24eadd9ec4416e"},
		{name: "input_591kb", seed: 7994582440685163287, dataSize: 606183, expectedHash: "b91c4de42b55bda52234042289b3bf3d1734a5d75a5a18611155aff228f46ea1"},
		{name: "input_835kb", seed: 6443739379063817906, dataSize: 855183, expectedHash: "9c75ce3e5b5aa9b50e0113461a7682483b8be5f3f20ead5624f1d1408609e434"},
		{name: "input_981kb", seed: 8211857585299887830, dataSize: 1005137, expectedHash: "3821c6e937883d314f16e338fd18e35b83f1799a40c3c2d4c32beba38be2fa86"},
		{name: "input_949kb", seed: 4988602387584303978, dataSize: 972492, expectedHash: "4ba10d3370e03f50e728778a3cf2987067510f236e41b11369bcaec7c67045ed"},
		{name: "input_529kb", seed: 908393246387930417, dataSize: 542428, expectedHash: "5ecc6aa26c56afeb33a08bdc342a12457ce31d64b4e912c6780b1499606025f0"},
		{name: "input_841kb", seed: 8063729658764635782, dataSize: 861755, expectedHash: "2af114cc248fdc5a4ca92f01478aed2a2b7173f16cb166ea399da9c73ef47fd8"},
		{name: "input_975kb", seed: 8890970237871203352, dataSize: 999011, expectedHash: "5f312d75503b13f3b9c45b4fdc154bca36bb86f57c5d7a7f8f284c08692f424b"},
		{name: "input_922kb", seed: 6775412096788225757, dataSize: 944895, expectedHash: "b31aa6b172041bc1c62bd1f1772edeac5f25d9b0a9cb31108bbc54ad36f0b81c"},
		{name: "input_679kb", seed: 6520300896597308550, dataSize: 695533, expectedHash: "d72bd8bad5dab66030de20d4d6e445ad9f6618fc26908bb67f242871d80033cd"},
		{name: "input_639kb", seed: 1441509335434118525, dataSize: 654427, expectedHash: "9a52759b63a035386fa557d03864748a59a6fe32f23b6c9e4073f90de098b673"},
		{name: "input_154kb", seed: 7323499654677190571, dataSize: 157750, expectedHash: "7b7b5a06b02a36752fe4ec4466fbd5512702e05a26ba4d931ce35270748bacfc"},
		{name: "input_920kb", seed: 9164370140782074545, dataSize: 942159, expectedHash: "f3389f2a85d227dc79b1e2f36ab1e7d8508da1cf148ff971d01dcb561973777e"},
		{name: "input_659kb", seed: 1807818337367627932, dataSize: 675228, expectedHash: "4610dbd541c8df88c5b6febaf77c2adca3ae5d582295b6fc89c1247396d78fe3"},
		{name: "input_448kb", seed: 4879942493625823277, dataSize: 459665, expectedHash: "d9402407f936db913a37562cce613c9a4ac6d1406540ef1b25ca413d3be55d4c"},
		{name: "input_345kb", seed: 8781143854627702169, dataSize: 353847, expectedHash: "66f563948a0ce88fdbaebac2ab6974aa8ce6c4f240bf6335bc8e688fcfcdb119"},
		{name: "input_948kb", seed: 5342241822494793137, dataSize: 971593, expectedHash: "27d60e3df9b8b22d1a268fff7eecfe5936a3ce2f200c2e0cbb393ff7905f1ce9"},
		{name: "input_268kb", seed: 9057688761589453769, dataSize: 274670, expectedHash: "8642b7f06781443234785b3aea76e994a43f4a875962958f4af869218630b878"},
		{name: "input_430kb", seed: 1283263450955133978, dataSize: 440780, expectedHash: "1b40388d701893ad9b7ef6b5e8144d6cace55eb9c16d79ea11491a82f6837ecb"},
		{name: "input_739kb", seed: 8464546535989325344, dataSize: 756910, expectedHash: "f03d1a4c36d87aab2f05e30cdd0fa3779a6d635487aa9d2651b3a9b37fc1b7d3"},
		{name: "input_501kb", seed: 3034897366669354117, dataSize: 513239, expectedHash: "ac1e62e7146fe1554a7d0c9c4a1205a61f54dad790e155c2ed90433230769611"},
		{name: "input_533kb", seed: 5886833906688734311, dataSize: 545987, expectedHash: "a83eed8b074306bc513a357a9d602e1714177d49cdbd37fe9121f54c983f6389"},
		{name: "input_561kb", seed: 6025688929181751162, dataSize: 574615, expectedHash: "4281155b8c253d0ca8ce6e5d7dbfd5326ce2fb58afa697e74bb1ea740a73131e"},
		{name: "input_166kb", seed: 2316217535716785625, dataSize: 170921, expectedHash: "74fbaf36955ee53af59b5b3690a7063c64538fa957e2b67f9acd5395b065602f"},
		{name: "input_353kb", seed: 6823168349905021995, dataSize: 361601, expectedHash: "c097863f4d44c8d8cb0ec0b5247fc73eac917efdeccab50f6cfb6b7b8a520c65"},
		{name: "input_585kb", seed: 13013938835543503, dataSize: 599144, expectedHash: "6f600e831cf1f2fe6e984ef5fd300c8debea983d69f6568d36c1dced44f72928"},
		{name: "input_553kb", seed: 7768111407243225335, dataSize: 567143, expectedHash: "6c5818bd7503ff603372f81ef299954e349cbdc366d8ad5bef0ea3578d866940"},
		{name: "input_516kb", seed: 771530412271058804, dataSize: 529160, expectedHash: "6fe8b0ef0ece561a8ba6375f5eb91a78450fd1bf7b5b89e967ec1cd9df60d4d3"},
		{name: "input_260kb", seed: 6332339359204160455, dataSize: 266612, expectedHash: "41d960c9bcca52fed7593b0abf62d1cf7a297b35e3630d1408b3e4e7a38d34de"},
		{name: "input_206kb", seed: 9141870321181686411, dataSize: 211830, expectedHash: "4df5f22b9d3da0cf35eb177ef38d1db849ce92f80ef7385f38d4e6957f16749c"},
		{name: "input_467kb", seed: 3968454238659711532, dataSize: 478567, expectedHash: "77a7498b518e8cd27799514be8ab33ea2c1214217a7adbd25216f20e346f1fb1"},
		{name: "input_257kb", seed: 8576527141580204154, dataSize: 263615, expectedHash: "d071ad70c5b031be67b302ec29e8f72aec6a9fd28862a84e54b887759fef4719"},
		{name: "input_657kb", seed: 4441891809806786300, dataSize: 672880, expectedHash: "47f1f26999dda76c15c63e0c0119c8a087e58f6a00eeecfeb6e472c3412cf0e2"},
		{name: "input_919kb", seed: 2318279817119990980, dataSize: 941206, expectedHash: "3c123a4ca8821d10f0d9fadd724065669aab522debac345dedf80fa3ccb50b39"},
		{name: "input_769kb", seed: 3543037439332401303, dataSize: 787836, expectedHash: "854c4b25546cac1fd957856a9a3a79119a212e7b5f8ed1d9c59a74adad103d10"},
		{name: "input_214kb", seed: 1438848673159967689, dataSize: 219171, expectedHash: "5d6f3ebf276f9c167c2074b9949d3beda0897dfc5c32803c461e4d04163c3c44"},
		{name: "input_775kb", seed: 1426967834779976657, dataSize: 793690, expectedHash: "bf4c24ade31c8efee31d24820baf04187535b93d17d64656a07c434031d18349"},
		{name: "input_348kb", seed: 6303393457478660289, dataSize: 356922, expectedHash: "8e72dc57e85f63655808c5292d2d8cbb1bf3630638b85a548b05b71952a5a9ce"},
		{name: "input_881kb", seed: 3846685509774361260, dataSize: 902541, expectedHash: "6f468b717cb6e88b32ae54845780ff16ed1a0e90afa6559069eb6073a4b65e61"},
		{name: "input_347kb", seed: 8646707523083679814, dataSize: 355528, expectedHash: "f2a964b156c1739713c2d1f924d068e429eeb8bbe217011d0b4ef2fdd9e7534b"},
		{name: "input_372kb", seed: 7793918982447394205, dataSize: 381836, expectedHash: "bcf9c2fb73b0fb62d491d65ccdf184f996d64cac131dbdc0f69ebb0d5d39acb9"},
		{name: "input_974kb", seed: 1034491783883478295, dataSize: 997604, expectedHash: "265dacf6ca42ef0da6daa06ed1d0baf7d078ecca66aafce5aad782c7c0aa9fda"},
		{name: "input_623kb", seed: 3910358027411881903, dataSize: 638751, expectedHash: "6709cd8a57771c8eebd56a78429bda0fc0d9b3a3c09d58b0b88d57cfb27da92e"},
		{name: "input_135kb", seed: 4273864263304037118, dataSize: 138695, expectedHash: "246ee3641be78854a887e8ede688390486cec115fecf8570114799f680376295"},
		{name: "input_884kb", seed: 5888461606762344739, dataSize: 905778, expectedHash: "8db58eae566a0fbfd4e4960eec9e44af01fb1b71c59a3b33619e3cce40aa4d9a"},
		{name: "input_154kb", seed: 3876831120760146157, dataSize: 158149, expectedHash: "008f2de98780136fc51d2a1dbd73ad6897ed0944acefbece5bce962941294089"},
		{name: "input_246kb", seed: 573424493786937151, dataSize: 252400, expectedHash: "06954e59b72a8b9d6a354f9866be3f07fbefebc5d1aadfe440ce9f4b70e86da3"},
		{name: "input_594kb", seed: 4413944639770035610, dataSize: 608645, expectedHash: "b5301ae85a8cb33a4033362336dc70f34884bb16a9ad22ee34ddbf330ac8cd83"},
		{name: "input_345kb", seed: 3413897104176568630, dataSize: 353309, expectedHash: "b7b23503916a89a7957e4be78a71459aece267a82cd86d8a9811db76d2ca00d5"},
		{name: "input_182kb", seed: 4687056650370225666, dataSize: 186578, expectedHash: "f128a43829888be66a4ad018542b91bb0b0d62240c867f57b8d8676b0da5c9ac"},
		{name: "input_596kb", seed: 7278932545205999809, dataSize: 610610, expectedHash: "11b0263d9feda6d4a5b23293ab69bb5949801707cbeb96ac7407f76069e43b82"},
		{name: "input_624kb", seed: 4847255447866565387, dataSize: 639996, expectedHash: "0227479f2e55c13b4349e10883ddefdbcc9048728d51a087d5eea38d315d3043"},
		{name: "input_425kb", seed: 7624391384633336572, dataSize: 436210, expectedHash: "84cbd23052fa9d386a721a2b1365cf532b082c36237deadab89da001f096e089"},
		{name: "input_867kb", seed: 7756330699333470086, dataSize: 887845, expectedHash: "26b3214d31e5c960611c684e923dcbb0a0491357698cb32cb4a275079d69c3bc"},
		{name: "input_981kb", seed: 7310704740221835421, dataSize: 1004964, expectedHash: "982d11446990e1982d58a5ed12c4af86fecbb68836a4aca1b4b9c8b3b9859276"},
		{name: "input_295kb", seed: 2333463841679714568, dataSize: 302261, expectedHash: "253300ab6ebfcc221c6b1afe2a4978d7b255d59df00dfc28ca73260bc546b3c9"},
		{name: "input_734kb", seed: 8088993938139972434, dataSize: 752243, expectedHash: "4af4b1c7c33f42ebe5752ed605537e79f529485baffab1bdbbb58040574b5e23"},
		{name: "input_153kb", seed: 1066889861504680337, dataSize: 157076, expectedHash: "7c02059b7a424f86af156c75d40f267147cc9c28192dfd2f13b32a494458244f"},
		{name: "input_247kb", seed: 5437576861888721920, dataSize: 253043, expectedHash: "ac8a9bb9ba3b1bc30943c577174d983afbf80b5c605b060f019041bab7a1425a"},
		{name: "input_884kb", seed: 8197742472884175475, dataSize: 905710, expectedHash: "3c1fbee1e1b0e2301d4f4d9110244919b2e99a5b9ff5d1254e62381ca136de27"},
		{name: "input_793kb", seed: 8351018417258750271, dataSize: 812126, expectedHash: "05baaf58273e114e5ca36ee64bdac76b313adca5cd1a0e0fbfca4d6dcad75c01"},
		{name: "input_312kb", seed: 516354799958707899, dataSize: 320233, expectedHash: "55b37702e4f99af44629480e8cfba15ea82b4602a5e61361335e3c95886406bd"},
		{name: "input_188kb", seed: 1258296322362886354, dataSize: 193369, expectedHash: "3eb9181992a21500975a61c0bf1178155df4d9c2ced30429232edf7dae3ef7d3"},
		{name: "input_405kb", seed: 2102928227887134539, dataSize: 415647, expectedHash: "2c354245d44205163fb3393547544e21c0cb3503ef46c21521edc1c8393a2e56"},
		{name: "input_892kb", seed: 4518193184260098255, dataSize: 913764, expectedHash: "cbb65ca3f083762b3b7e831de3cc89f3f4e165fcb79cf5a777f70ea4804c7d9f"},
		{name: "input_806kb", seed: 2770680402710233691, dataSize: 826157, expectedHash: "c85b2703c72db6ef892a994434b6764fb0441d74b6f1e1abe2724882b4c2768e"},
		{name: "input_917kb", seed: 2814033802645763477, dataSize: 939058, expectedHash: "142fc5a05f8012a9f45fe243792d88ea12d7bbc2f120c8437cecffb416c72835"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := generateTestInput(tc.seed, tc.dataSize)
			r := io.NopCloser(bytes.NewReader(testData.data))
			compressedReader := ReadCloser(r)
			defer compressedReader.Close()

			hash, _, err := v1.SHA256(compressedReader)
			require.NoError(t, err)
			require.Equal(t, tc.expectedHash, hash.Hex)
		})
	}
}
